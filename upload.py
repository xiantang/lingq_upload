#!/usr/bin/env python


import argparse
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

load_dotenv()
key = os.getenv("APIKey")
postAddress = os.getenv("postAddress")
mypath = os.getenv("mypath")
status = os.getenv("status")


parser = argparse.ArgumentParser(description="a tool for Upload audio book to lingq.")
parser.add_argument("-a", "--audio_folder")
parser.add_argument("-b", "--book_path")
parser.add_argument("-t", "--title")
args = parser.parse_args()


if not (args.audio_folder or args.book_path or args.title):
    parser.error("No action requested, add --audio_folder or --book_path or --title")


header = {"Authorization": key, "Content-Type": "application/json"}

title = args.title
discriprtion = """
"""

book = epub.read_epub(args.book_path)

cover = glob(args.audio_folder + "/*.jpg")


def chapter_to_str(doc):
    soup = BeautifulSoup(doc.content, "html.parser")
    text = [para.get_text() for para in soup.find_all("p")]
    a = "\r\n\r\n".join(text)
    return a


def create_collections(
    title,
    description,
):
    url = "https://www.lingq.com/api/v3/en/collections/"
    body = {
        "description": description,
        "hasPrice": False,
        "isFeatured": False,
        "sourceURLEnabled": False,
        "language": "en",
        "level": 1,
        "sellAll": False,
        "tags": [],
        "title": title,
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
    url = "https://www.lingq.com/api/v3/en/collections/" + str(collectonID)
    r = requests.patch(
        url=url,
        data=m,
        headers=h,
    )


def upload_lessons(collectionID):
    listofmp3s = glob(args.audio_folder + "/*.mp3")
    listofmp3s.sort()
    list_book_charpter = []
    for c in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        if "split" in c.get_name():
            list_book_charpter.append(c)

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
        r = requests.post(postAddress, json=body, headers=h)
        lesson_id = r.json()["id"]
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


collectionID = create_collections(title, discriprtion)
if len(cover) > 0:
    upload_cover(cover[0], collectionID)

upload_lessons(collectionID)

generate_timestamp_for_course(collectionID)
