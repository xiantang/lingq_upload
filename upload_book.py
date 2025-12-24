#!/usr/bin/env python
import argparse
import json
import logging
import os
import sys
from glob import glob
from os.path import basename

import ebooklib
import requests
from bs4 import BeautifulSoup
from dotenv import load_dotenv
from ebooklib import epub
from requests_toolbelt.multipart.encoder import MultipartEncoder

from generate_timestamp import generate_timestamp_for_course
from update_lesson import update_metadata

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

load_dotenv()
key = os.getenv("APIKey")
postAddress = os.getenv("postAddress")
status = os.getenv("status")

parser = argparse.ArgumentParser(
    description="Upload audio books to LingQ from a structured directory.",
    epilog="Example: python3 upload_book.py downloads/my-book -v"
)

# Position argument (optional, mutually exclusive with -d)
parser.add_argument(
    "directory",
    nargs="?",
    help="Directory containing book files (EPUB, MP3s, metadata.json)"
)

# Named argument (mutually exclusive with position argument)
parser.add_argument(
    "-d", "--dir",
    dest="directory_named",
    help="Directory containing book files (alternative to positional argument)"
)

# Optional override parameters
parser.add_argument(
    "--title",
    help="Override title from metadata.json"
)

parser.add_argument(
    "--level",
    choices=["Beginner 1", "Beginner 2", "Intermediate 1", "Intermediate 2", "Advanced 1", "Advanced 2"],
    help="Override level from metadata.json"
)

parser.add_argument(
    "--tags",
    help="Override tags from metadata.json (comma-separated)"
)

parser.add_argument(
    "-v", "--verbose",
    action="store_true",
    help="Enable verbose debug logging"
)

args = parser.parse_args()

level_mapping = {
    "Beginner 1": 1,
    "Beginner 2": 2,
    "Intermediate 1": 3,
    "Intermediate 2": 4,
    "Advanced 1": 5,
    "Advanced 2": 6,
}


header = {"Authorization": key, "Content-Type": "application/json"}


def discover_book_files(directory):
    """
    Auto-discover book files in directory.
    
    Returns:
        dict with keys: 'epub', 'mp3s', 'cover', 'metadata'
        
    Raises:
        FileNotFoundError: if required files are missing
    """
    result = {
        'epub': None,
        'mp3s': [],
        'cover': None,
        'metadata': None
    }
    
    dirname = os.path.basename(directory)
    
    # 1. Find metadata.json (REQUIRED)
    metadata_path = os.path.join(directory, "metadata.json")
    if not os.path.exists(metadata_path):
        raise FileNotFoundError(
            f"metadata.json not found in {directory}. "
            f"This file is required for upload."
        )
    result['metadata'] = metadata_path
    logger.debug(f"Found metadata.json: {metadata_path}")
    
    # 2. Find EPUB file (REQUIRED)
    epub_files = glob(os.path.join(directory, "*.epub"))
    if not epub_files:
        raise FileNotFoundError(f"No EPUB file found in {directory}")
    if len(epub_files) > 1:
        logger.warning(f"Multiple EPUB files found, using first: {epub_files[0]}")
    result['epub'] = epub_files[0]
    logger.debug(f"Found EPUB: {os.path.basename(result['epub'])}")
    
    # 3. Find MP3 files (REQUIRED)
    # Priority 1: Root directory
    mp3_files = sorted(glob(os.path.join(directory, "*.mp3")))
    
    # Priority 2: <dirname>_splitted subdirectory
    if not mp3_files:
        splitted_dir = os.path.join(directory, f"{dirname}_splitted")
        if os.path.isdir(splitted_dir):
            mp3_files = sorted(glob(os.path.join(splitted_dir, "*.mp3")))
            logger.info(f"Found MP3s in _splitted subdirectory: {len(mp3_files)} files")
    
    if not mp3_files:
        raise FileNotFoundError(
            f"No MP3 files found in {directory} or {dirname}_splitted/. "
            f"If you have a single MP3 + CUE file, please split it first."
        )
    
    # Filter out very large files (likely unsplit audio)
    filtered_mp3s = []
    for mp3 in mp3_files:
        size_mb = os.path.getsize(mp3) / (1024 * 1024)
        if size_mb > 100:  # Assume files >100MB are unsplit
            logger.warning(
                f"Skipping large MP3 file (likely unsplit): {os.path.basename(mp3)} ({size_mb:.1f}MB). "
                f"Please split this file into chapters first."
            )
        else:
            filtered_mp3s.append(mp3)
    
    if not filtered_mp3s:
        raise FileNotFoundError(
            f"No valid MP3 chapter files found. Found only large unsplit files. "
            f"Please split your audio into chapters."
        )
    
    result['mp3s'] = filtered_mp3s
    logger.debug(f"Found {len(filtered_mp3s)} MP3 chapter files")
    
    # 4. Find cover image (OPTIONAL)
    # Priority 1: Root directory
    for ext in ['jpg', 'jpeg', 'png']:
        cover_pattern = os.path.join(directory, f"cover.{ext}")
        cover_files = glob(cover_pattern)
        if cover_files:
            result['cover'] = cover_files[0]
            logger.debug(f"Found cover: {os.path.basename(result['cover'])}")
            break
    
    # Priority 2: _splitted subdirectory
    if not result['cover']:
        splitted_dir = os.path.join(directory, f"{dirname}_splitted")
        if os.path.isdir(splitted_dir):
            for ext in ['jpg', 'jpeg', 'png']:
                cover_pattern = os.path.join(splitted_dir, f"cover.{ext}")
                cover_files = glob(cover_pattern)
                if cover_files:
                    result['cover'] = cover_files[0]
                    logger.debug(f"Found cover in _splitted: {os.path.basename(result['cover'])}")
                    break
    
    # Priority 3: Extract from EPUB (handled later)
    if not result['cover']:
        logger.debug("No cover file found, will attempt EPUB extraction")
    
    return result


def load_metadata(metadata_path, overrides=None):
    """
    Load and validate metadata.json.
    
    Args:
        metadata_path: Path to metadata.json
        overrides: Dict with optional 'title', 'level', 'tags' to override
        
    Returns:
        dict with keys: 'title', 'description', 'level', 'tags', 'author'
    """
    logger.info("Loading metadata from metadata.json")
    
    with open(metadata_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # Extract fields with defaults
    metadata = {
        'title': data.get('title', ''),
        'description': data.get('description', ''),
        'level': data.get('level', ''),
        'tags': data.get('tags', []),
        'author': data.get('author', ''),
    }
    
    # Validate required fields
    if not metadata['title']:
        raise ValueError("metadata.json missing required field: 'title'")
    
    # Clean tags (remove HTML entities, limit to 9)
    cleaned_tags = []
    for tag in metadata['tags'][:9]:  # Max 9 (we add "book" later = 10 total)
        clean_tag = tag.replace("&nbsp;", "").replace("&amp;", "&").strip()
        if clean_tag:
            cleaned_tags.append(clean_tag)
    metadata['tags'] = cleaned_tags
    
    # Apply command-line overrides
    if overrides:
        if overrides.get('title'):
            logger.info(f"Overriding title: {overrides['title']}")
            metadata['title'] = overrides['title']
        
        if overrides.get('level'):
            logger.info(f"Overriding level: {overrides['level']}")
            metadata['level'] = overrides['level']
        
        if overrides.get('tags'):
            # Parse comma-separated tags
            override_tags = [t.strip() for t in overrides['tags'].split(',') if t.strip()]
            logger.info(f"Overriding tags: {override_tags}")
            metadata['tags'] = override_tags[:9]
    
    logger.debug(f"Loaded metadata - Title: '{metadata['title']}', Level: '{metadata['level']}', Tags: {len(metadata['tags'])}")
    
    return metadata


def chapter_to_str(doc):
    soup = BeautifulSoup(doc.content, "html.parser")
    text = [para.get_text() for para in soup.find_all("p")]
    a = "\r\n\r\n".join(text)
    return a


def extract_cover_from_epub(book_obj, output_dir):
    """
    Extract cover image from EPUB object
    
    Args:
        book_obj: ebooklib EPUB object
        output_dir: Directory to save the cover image
        
    Returns:
        str: Path to extracted cover image, or None if not found
    """
    try:
        # Iterate through all image items in the EPUB
        for item in book_obj.get_items_of_type(ebooklib.ITEM_IMAGE):
            item_name = item.get_name().lower()
            
            # Look for images with 'cover' in the filename
            if 'cover' in item_name:
                # Determine file extension
                if item_name.endswith('.jpg') or item_name.endswith('.jpeg'):
                    ext = 'jpg'
                elif item_name.endswith('.png'):
                    ext = 'png'
                else:
                    ext = 'jpg'  # Default to jpg
                
                # Save the cover image
                cover_path = os.path.join(output_dir, f'cover.{ext}')
                with open(cover_path, 'wb') as f:
                    f.write(item.content)
                
                logger.info(f"Extracted cover from EPUB: {item.get_name()}")
                return cover_path
        
        # No cover found
        return None
        
    except Exception as e:
        logger.warning(f"Failed to extract cover from EPUB: {e}")
        return None


def create_collections(title, description, tags, level, sourceURL):
    url = "https://www.lingq.com/api/v3/en/collections/"
    tags.append("book")
    body = {
        "description": description,
        "hasPrice": False,
        "isFeatured": False,
        "sourceURLEnabled": False,
        "language": "en",
        "level": level_mapping.get(level, 1),
        "sellAll": False,
        "tags": tags,
        "title": title,
        "sourceURL": sourceURL,
    }
    r = requests.post(
        url,
        json=body,
        headers=header,
    )
    return r.json()["id"]


def upload_cover(cover_path, collectonID):
    logger.info("Uploading cover image to collection")
    m = MultipartEncoder(
        [
            ("image", (cover_path, open(cover_path, "rb"), "image/jpg")),
        ]
    )
    h = {"Authorization": key, "Content-Type": m.content_type}
    url = "https://www.lingq.com/api/v3/en/collections/" + str(collectonID) + "/"
    r = requests.patch(
        url=url,
        data=m,
        headers=h,
    )


def upload_lessons(collectionID, book, listofmp3s, cover):
    """
    Upload lessons to LingQ collection.
    
    Args:
        collectionID: LingQ collection ID
        book: EPUB book object
        listofmp3s: List of MP3 file paths
        cover: List with cover path (or empty list)
    """
    # First, try to find chapters with "split" in the name (pre-split EPUBs)
    list_book_charpter = []
    for c in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        if "split" in c.get_name():
            list_book_charpter.append(c)

    # If no split chapters found, use all content documents (single-file EPUBs)
    # Exclude titlepage to avoid uploading non-content pages
    if len(list_book_charpter) == 0:
        logger.info("No split chapters found, treating as single-file EPUB")
        for c in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
            name_lower = c.get_name().lower()
            # Exclude titlepage, but include index/content files
            if "title" not in name_lower or "index" in name_lower:
                list_book_charpter.append(c)

    logger.debug(f"Found {len(listofmp3s)} MP3 files and {len(list_book_charpter)} chapters")
    
    # Validate that we have chapters
    if len(list_book_charpter) == 0:
        raise Exception("Sorry, no valid chapters found in EPUB. The EPUB may be empty or corrupted.")
    
    # Validate that chapter count matches MP3 count (strict mode)
    if len(list_book_charpter) != len(listofmp3s):
        raise Exception(
            f"Chapter count ({len(list_book_charpter)}) must match MP3 count ({len(listofmp3s)}). "
            f"Please ensure you have the same number of text chapters and audio files."
        )

    logger.info(f"Starting upload of {len(listofmp3s)} lessons")
    
    for idx, (doc, audiofile) in enumerate(list(zip(list_book_charpter, listofmp3s)), 1):
        s = chapter_to_str(doc)
        mp3name = basename(audiofile)
        title = mp3name.split(".")[0]
        logger.info(f"Creating lesson {idx}/{len(listofmp3s)}: {title}")
        body = {
            "title": title,
            "status": status,
            "collection": collectionID,
            "text": s,
        }
        h = {"Authorization": key, "Content-Type": "application/json"}
        # Use v3 API endpoint (v2 is obsolete for POST)
        lesson_endpoint = postAddress.replace("/v2/", "/v3/")
        r = requests.post(lesson_endpoint, json=body, headers=h)
        response_data = r.json()
        
        # Handle API response - check if it's a dict with 'id' key
        if isinstance(response_data, dict) and "id" in response_data:
            lesson_id = response_data["id"]
            logger.debug(f"Lesson created successfully (ID: {lesson_id})")
        else:
            logger.error(f"Failed to create lesson - API response: {response_data}")
            raise Exception(f"Failed to create lesson. API response: {response_data}")
        
        logger.info(f"Uploading audio file: {basename(audiofile)}")
        body = [
            ("language", "en"),
            ("audio", (audiofile, open(audiofile, "rb"), "audio/mpeg")),
        ]
        if len(cover) > 0:
            body.append(("image", (cover[0], open(cover[0], "rb"), "image/jpg")))

        m = MultipartEncoder(body)
        h = {"Authorization": key, "Content-Type": m.content_type}
        r = requests.patch(
            "https://www.lingq.com/api/v3/en/lessons/" + str(lesson_id) + "/",
            data=m,
            headers=h,
        )


if __name__ == "__main__":
    # Set log level based on verbose flag
    if args.verbose:
        logger.setLevel(logging.DEBUG)
    
    # Get directory (from positional or named argument)
    directory = args.directory or args.directory_named
    if not directory:
        parser.error("Directory is required. Usage: upload_book.py <directory> or upload_book.py -d <directory>")
    
    # Validate directory exists
    if not os.path.isdir(directory):
        logger.error(f"Directory not found: {directory}")
        sys.exit(1)
    
    directory = os.path.abspath(directory.rstrip('/'))
    logger.info(f"Processing directory: {directory}")
    
    try:
        # Step 1: Discover files
        logger.info("Discovering book files...")
        files = discover_book_files(directory)
        
        # Step 2: Load metadata
        overrides = {
            'title': args.title,
            'level': args.level,
            'tags': args.tags,
        }
        metadata = load_metadata(files['metadata'], overrides)
        
        # Step 3: Load EPUB
        logger.info(f"Reading EPUB: {os.path.basename(files['epub'])}")
        book = epub.read_epub(files['epub'])
        
        # Step 4: Handle cover
        cover_path = files['cover']
        if not cover_path:
            logger.info("No cover file found, attempting EPUB extraction")
            cover_path = extract_cover_from_epub(book, directory)
            if cover_path:
                logger.info(f"Extracted cover from EPUB: {os.path.basename(cover_path)}")
            else:
                logger.warning("No cover image available, will skip cover upload")
        
        # Convert to list for compatibility with existing code
        cover = [cover_path] if cover_path else []
        
        # Step 5: Prepare variables for upload
        title = metadata['title']
        description = metadata['description']
        level = metadata['level']
        tags = metadata['tags']
        listofmp3s = files['mp3s']
        
        # Step 6: Create collection
        logger.info(f"Creating collection: {title}")
        collectionID = create_collections(
            title, description, tags, level, "https://english-e-reader.net"
        )
        logger.info(f"Collection created (ID: {collectionID})")
        
        # Step 7: Upload cover
        if cover:
            upload_cover(cover[0], collectionID)
        
        # Step 8: Upload lessons
        upload_lessons(collectionID, book, listofmp3s, cover)
        logger.info("All lessons uploaded successfully")
        
        # Step 9: Update metadata
        logger.info("Updating collection metadata")
        update_metadata(collectionID, tags, level_mapping.get(level, 1))
        
        # Step 10: Generate timestamps
        logger.info("Generating timestamps for lessons")
        generate_timestamp_for_course(collectionID)
        
        logger.info("Book upload completed successfully")
        
    except FileNotFoundError as e:
        logger.error(f"File discovery failed: {e}")
        sys.exit(1)
    except ValueError as e:
        logger.error(f"Metadata validation failed: {e}")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Upload failed: {e}")
        raise
