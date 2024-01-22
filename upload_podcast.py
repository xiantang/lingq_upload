import os
from glob import glob

from dotenv import load_dotenv
from pydub import AudioSegment

load_dotenv()

# from openai import OpenAI

# client = OpenAI()
#
# audio_file = open("/path/to/file/audio.mp3", "rb")
# transcript = client.audio.transcriptions.create(model="whisper-1", file=audio_file)


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
        chunk.export(f"luke_back/{title}-{i}.mp3", format="mp3")


for mp3 in newmp3s:
    try:
        processing(mp3)
    except Exception as error:
        # handle the exception
        print(
            "An exception occurred:", error
        )  # An exception occurred: division by zero
