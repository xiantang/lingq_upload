import os

import requests
from dotenv import load_dotenv

load_dotenv()
key = os.getenv("APIKey")
header = {"Authorization": key, "Content-Type": "application/json"}


def generate_timestamp(lesson_id):
    print("generating timestamp..." + " " + str(lesson_id))
    r = requests.post(
        "https://www.lingq.com/api/v3/en/lessons/" + str(lesson_id) + "/genaudio/",
        json={},
        headers=header,
    )
    if r.status_code == 200:
        print("generate_successed")


def get_lessons(collectonID):
    url = (
        "https://www.lingq.com/api/v3/en/collections/"
        + str(collectonID)
        + "/lessons/?page=1&page_size=100&sortBy=pos"
    )
    r = requests.get(
        url,
        headers=header,
    )
    return r.json()


def generate_timestamp_for_course(collectonID):
    lessons = get_lessons(collectonID)
    for result in lessons["results"]:
        lesson_id = result["id"]
        generate_timestamp(lesson_id)
