import os
from glob import glob

from dotenv import load_dotenv
from openai import OpenAI
from pydub import AudioSegment

client = OpenAI(api_key="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")

load_dotenv()


folder = "luke/*.mp3"
listofmp3s = glob(folder)
newmp3s = listofmp3s[5:390]


def processing(mp3):
    basename = os.path.basename(mp3)
    title = basename.replace(".mp3", "")
    print(title)
    audio = AudioSegment.from_mp3(mp3)
    chunk_length = 1000 * 1000  # in milliseconds
    chunks = [audio[i : i + chunk_length] for i in range(0, len(audio), chunk_length)]
    # split mp3
    for i, chunk in enumerate(chunks):
        t = f"luke_back/{title}-{i}.mp3"
        chunk.export(t, format="mp3")
    chunks = glob(f"luke_back/{title}-*.mp3")
    text = ""
    for chunk in chunks:
        print(chunk)
        audio_file = open(chunk, "rb")
        transcript = client.audio.transcriptions.create(
            model="whisper-1", file=audio_file
        )
        print(transcript)
        text = text + transcript.text
    print(11)
    print(text)

    # print(chunks)


for mp3 in newmp3s:
    try:
        processing(mp3)
    except Exception as error:
        # handle the exception
        print(
            "An exception occurred:", error
        )  # An exception occurred: division by zero
