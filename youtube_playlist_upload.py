import os
import re
from glob import glob
from os.path import basename

import requests
from dotenv import load_dotenv
from requests_toolbelt.multipart.encoder import MultipartEncoder

from generate_timestamp import generate_timestamp

folder = "/home/neo/project/test"
listofmp3s = glob(folder + "**/*.mp3")

# print(listofmp3s)

collectionID = "1559870"
load_dotenv()
listofmp3s.sort()


def num_sort(test_string):
    return list(map(int, re.findall(r"\d+", test_string)))[0]


listofmp3s.sort(key=num_sort)

for mp3 in listofmp3s:
    mp3name = basename(mp3)
    title = mp3name[:-4]
    t = title.split("-")[0].replace("_", " ")
    subtitle = title + ".en.srt"
    body = [
        ("language", "en"),
        ("collection", str(collectionID)),
        ("isHidden", "true"),
        ("title", t),
        ("save", "true"),
        ("audio", (mp3name, open(mp3, "rb"), "audio/mpeg")),
        ("file", (subtitle, open(folder + "/" + subtitle, "rb"), "text/srt")),
    ]
    key = os.getenv("APIKey")
    #
    m = MultipartEncoder(body)
    h = {
        "Authorization": key,
        "Content-Type": m.content_type,
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    }

    r = requests.post(
        "https://www.lingq.com/api/v3/en/lessons/import/", data=m, headers=h
    )
    print(r.text)
    print(r.json())
    lesson_id = r.json()["id"]
    print("uploading audiofile...")
    generate_timestamp(lesson_id)
