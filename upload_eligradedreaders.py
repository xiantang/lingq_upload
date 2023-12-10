import json
import os
from glob import glob
from os.path import basename

import requests
from requests_toolbelt.multipart.encoder import MultipartEncoder

from upload_book import create_collections

name = "A Tale of Two Cities"

folder = name

title = ""
detail = ""
level = ""


with open(folder + "/" + name + ".json", "r") as file:
    content = file.read()
    data = json.loads(content)
    title = data["title"]
    detail = data["detail"]
    level = data["level"]

collectionID = create_collections(
    title, detail, [], level, "https://www.eligradedreaders.com"
)


listofmp3s = glob(folder + "**/*.mp3", recursive=True)

for mp3 in listofmp3s:
    mp3name = basename(mp3)
    body = [
        ("language", "en"),
        ("collection", str(collectionID)),
        ("isHidden", "true"),
        ("title", mp3name),
        ("save", "true"),
        ("audio", (mp3name, open(mp3, "rb"), "audio/mpeg")),
    ]
    key = os.getenv("APIKey")

    m = MultipartEncoder(body)
    h = {
        "Authorization": key,
        "Content-Type": m.content_type,
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    }

    r = requests.post(
        "https://www.lingq.com/api/v3/en/lessons/import/", data=m, headers=h
    )
    print(r.json())
    # lesson_id = r.json()["id"]
    # print("uploading audiofile...")

print(collectionID)
