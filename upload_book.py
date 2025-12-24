#!/usr/bin/env python
import argparse
import json
import os
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

load_dotenv()
key = os.getenv("APIKey")
postAddress = os.getenv("postAddress")
status = os.getenv("status")

parser = argparse.ArgumentParser(description="a tool for Upload audio book to lingq.")
parser.add_argument("-a", "--audio_folder")
parser.add_argument("-b", "--book_path")
parser.add_argument("-t", "--title")
parser.add_argument("-f", "--folder")
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


def chapter_to_str(doc):
    soup = BeautifulSoup(doc.content, "html.parser")
    text = [para.get_text() for para in soup.find_all("p")]
    a = "\r\n\r\n".join(text)
    return a


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
    print("uploading cover ...")
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


def upload_aduios(collectionID):
    # First, try to find chapters with "split" in the name (pre-split EPUBs)
    list_book_charpter = []
    for c in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        if "split" in c.get_name():
            list_book_charpter.append(c)

    # If no split chapters found, use all content documents (single-file EPUBs)
    # Exclude titlepage to avoid uploading non-content pages
    if len(list_book_charpter) == 0:
        print("No split chapters found, treating as single-file EPUB")
        for c in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
            name_lower = c.get_name().lower()
            # Exclude titlepage, but include index/content files
            if "title" not in name_lower or "index" in name_lower:
                list_book_charpter.append(c)

    print("len of mp3 " + str(len(listofmp3s)))
    print("len of chapter " + str(len(list_book_charpter)))
    
    # Validate that we have chapters
    if len(list_book_charpter) == 0:
        raise Exception("Sorry, no valid chapters found in EPUB. The EPUB may be empty or corrupted.")
    
    # Validate that chapter count matches MP3 count (strict mode)
    if len(list_book_charpter) != len(listofmp3s):
        raise Exception(
            f"Chapter count ({len(list_book_charpter)}) must match MP3 count ({len(listofmp3s)}). "
            f"Please ensure you have the same number of text chapters and audio files."
        )

    for doc, audiofile in list(zip(list_book_charpter, listofmp3s)):
        s = chapter_to_str(doc)
        mp3name = basename(audiofile)
        title = mp3name.split(".")[0]
        print("creating lesson " + title + " ...")
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
        print(response_data)
        
        # Handle API response - check if it's a dict with 'id' key
        if isinstance(response_data, dict) and "id" in response_data:
            lesson_id = response_data["id"]
        else:
            raise Exception(f"Failed to create lesson. API response: {response_data}")
        print("uploading audiofile...")
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
    if not (args.audio_folder or args.book_path or args.title or args.folder):
        parser.error(
            "No action requested, add --audio_folder or --book_path or --title"
        )

    title = args.title
    discriprtion = """
    """

    level = ""
    if args.folder:
        book = glob(args.folder + "/*.epub")
        book = epub.read_epub(book[0])
        cover_file = args.folder + "/" + args.folder + "_splitted/cover.jpg"
        cover = glob(cover_file)
        audio_files = args.folder + "/" + args.folder + "_splitted" + "/*.mp3"
        listofmp3s = glob(audio_files, recursive=True)
        listofmp3s.sort()
        with open(args.folder + "/metadata.json", "r") as file:
            file_content = file.read()  # Read the content of the file as a string
            data = json.loads(file_content)
            title = data["title"]
            discriprtion = data["description"]
            level = data["level"]
            t = []
            count = 0
            # because max is 10 tags
            for tag in data["tags"]:
                if count > 8:
                    break
                count += 1
                t.append(tag)
            tags = t
    else:
        book = epub.read_epub(args.book_path)
        listofmp3s = glob(args.audio_folder + "/*.mp3")
        cover = glob(args.audio_folder + "/*.jpg")
        tags = []
        
        # Check if metadata.json exists in the audio folder
        metadata_path = os.path.join(args.audio_folder, "metadata.json")
        if os.path.exists(metadata_path):
            print(f"Found metadata.json, loading metadata...")
            with open(metadata_path, "r") as file:
                data = json.loads(file.read())
                # Use metadata values, but allow command-line title to override
                if not args.title or args.title == data.get("title", ""):
                    title = data.get("title", title)
                discriprtion = data.get("description", discriprtion)
                level = data.get("level", level)
                # Get tags from metadata (max 10, but script already limits to 9)
                metadata_tags = data.get("tags", [])
                t = []
                count = 0
                for tag in metadata_tags:
                    if count > 8:  # max is 10, but we add "book" tag later
                        break
                    # Clean up HTML entities in tags
                    clean_tag = tag.replace("&nbsp;", "").strip()
                    if clean_tag:  # only add non-empty tags
                        t.append(clean_tag)
                        count += 1
                tags = t
                print(f"Loaded metadata: level={level}, tags count={len(tags)}")

    collectionID = create_collections(
        title, discriprtion, tags, level, "https://english-e-reader.net"
    )
    if len(cover) > 0:
        upload_cover(cover[0], collectionID)

    upload_aduios(collectionID)

    update_metadata(collectionID, tags, level_mapping.get(level, 1))
    generate_timestamp_for_course(collectionID)
