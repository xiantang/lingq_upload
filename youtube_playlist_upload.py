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

collectionID = "1568793"
load_dotenv()
listofmp3s.sort()


def num_sort(test_string):
    return list(map(int, re.findall(r"\d+", test_string)))[0]


listofmp3s.sort(key=num_sort)


def uploading(mp3):
    mp3name = basename(mp3)
    title = mp3name[:-4]
    t = title.split("-")[0].replace("_", " ")
    subtitle = title + ".en.srt"
    subtitle_path = folder + "/" + subtitle
    body = [
        ("language", "en"),
        ("collection", str(collectionID)),
        ("isHidden", "true"),
        ("title", t),
        ("save", "true"),
        ("audio", (mp3name, open(mp3, "rb"), "audio/mpeg")),
        ("file", (subtitle, open(subtitle_path, "rb"), "text/srt")),
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


for mp3 in listofmp3s:
    try:
        uploading(mp3)
    except Exception as error:
        # handle the exception
        print(
            "An exception occurred:", error
        )  # An exception occurred: division by zero

# yt-dlp -x --audio-format mp3 --convert-subs srt --write-auto-subs           --restrict-filenames     --playlist-reverse "https://www.youtube.com/watch?v=c6_aduAL25c&list=PLWYV8lHn0fV9rcRuxs_--mzLZewjziP4-&index=190"  --sub-format ttml --convert-subs srt --exec 'before_dl:fn=$(echo %(_filename)s| sed "s/%(ext)s/en.srt/g") && ffmpeg -fix_sub_duration -i "$fn" -c:s text "$fn".tmp.srt && mv "$fn".tmp.srt "$fn"'
