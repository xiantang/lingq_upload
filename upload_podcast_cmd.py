import argparse
import os
from glob import glob

import requests
from dotenv import load_dotenv
from pydub import AudioSegment
from requests_toolbelt.multipart.encoder import MultipartEncoder

parser = argparse.ArgumentParser(description="A tool for uploading a podcast to LingQ.")
parser.add_argument("-p", "--mp3_path", required=True, help="Path to the MP3 file.")
parser.add_argument("-t", "--title", required=True, help="Title of the podcast.")
parser.add_argument(
    "-d", "--description", required=False, help="Description of the podcast."
)
args = parser.parse_args()

load_dotenv()
key = os.getenv("APIKey")
# collectionID = "1696280"

level_mapping = {
    "Beginner 1": 1,
    "Beginner 2": 2,
    "Intermediate 1": 3,
    "Intermediate 2": 4,
    "Advanced 1": 5,
    "Advanced 2": 6,
}

header = {"Authorization": key, "Content-Type": "application/json"}


def create_collections(title, description, tags, level, sourceURL):
    url = "https://www.lingq.com/api/v3/en/collections/"
    if description == None:
        description = ""

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


collectionID = create_collections(
    args.title, args.description, ["Podcast"], "Intermediate 2", ""
)


def processing_without_transcript(mp3):
    basename = os.path.basename(mp3)
    title = basename.replace(".mp3", "")
    print(title)
    audio = AudioSegment.from_mp3(mp3)
    chunk_length = 600 * 1000  # in milliseconds
    chunks = [audio[i : i + chunk_length] for i in range(0, len(audio), chunk_length)]
    # split mp3
    print("start to split audio")
    for i, chunk in enumerate(chunks):
        t = f"luke_back/{title}-{i}.mp3"
        chunk.export(t, format="mp3")
    chunks = glob(f"luke_back/{title}-*.mp3")
    for chunk in chunks:
        newname = os.path.basename(chunk)
        title = newname.replace(".mp3", "")
        body = [
            ("language", "en"),
            ("collection", str(collectionID)),
            ("isHidden", "true"),
            ("title", title.replace("-", " ")),
            ("save", "true"),
            ("audio", (chunk, open(chunk, "rb"), "audio/mpeg")),
        ]
        m = MultipartEncoder(body)
        h = {
            "Authorization": key,
            "Content-Type": m.content_type,
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
        }

        r = requests.post(
            "https://www.lingq.com/api/v3/en/lessons/import/", data=m, headers=h
        )
        print("success " + title)
        print(r.text)


processing_without_transcript(args.mp3_path)
