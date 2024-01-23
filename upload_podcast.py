import os
import tempfile
import traceback
from glob import glob

import requests
from dotenv import load_dotenv
from openai import OpenAI
from pydub import AudioSegment
from requests_toolbelt.multipart.encoder import MultipartEncoder

from generate_timestamp import generate_timestamp

client = OpenAI(api_key="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")

load_dotenv()
key = os.getenv("APIKey")
collectionID = "1582106"


folder = "luke/*.mp3"
listofmp3s = glob(folder)
newmp3s = listofmp3s[:54]


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
        # lesson_id = r.json()["id"]


def processing(mp3):
    basename = os.path.basename(mp3)
    title = basename.replace(".mp3", "")
    print(title)
    audio = AudioSegment.from_mp3(mp3)
    chunk_length = 1000 * 1000  # in milliseconds
    chunks = [audio[i : i + chunk_length] for i in range(0, len(audio), chunk_length)]
    # split mp3
    print("start to split audio")
    for i, chunk in enumerate(chunks):
        t = f"luke_back/{title}-{i}.mp3"
        chunk.export(t, format="mp3")
    chunks = glob(f"luke_back/{title}-*.mp3")
    text = ""

    print("start to generate transcriptions, chunks: " + str(len(chunks)))
    for chunk in chunks:
        audio_file = open(chunk, "rb")
        transcript = client.audio.transcriptions.create(
            model="whisper-1", file=audio_file
        )
        text = text + transcript.text
        print("finished add transcript for" + chunk)
    sentences = text.split(".")
    text = "\n".join(
        sentence.strip() + "." for sentence in sentences if sentence.strip()
    )
    tmp = tempfile.NamedTemporaryFile()
    with open(tmp.name, "w") as f:
        f.write(text)
    print("generate text " + text)
    body = [
        ("language", "en"),
        ("collection", str(collectionID)),
        ("isHidden", "true"),
        ("title", title.replace("-", " ")),
        ("save", "true"),
        ("audio", (basename, open(mp3, "rb"), "audio/mpeg")),
        ("text", text),
        ("file", (tmp.name, open(tmp.name, "rb"), "text/plain")),
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
    print(r.text)
    lesson_id = r.json()["id"]
    print("uploading audiofile...")
    generate_timestamp(lesson_id)


if __name__ == "__main__":
    for mp3 in newmp3s:
        try:
            processing_without_transcript(mp3)
        except Exception as error:
            print(traceback.format_exc())
    print(collectionID)

    # generate_timestamp_for_course(collectionID)
