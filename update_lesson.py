import os

import requests
from dotenv import load_dotenv

from generate_timestamp import get_lessons

load_dotenv()
key = os.getenv("APIKey")


header = {"Authorization": key, "Content-Type": "application/json"}


def update_metadata(collectonID, tags, level):
    lessons = get_lessons(collectonID)
    lesson_ids = []
    for result in lessons["results"]:
        lesson_id = result["id"]
        lesson_ids.append(lesson_id)
    body = {
        "ids": lesson_ids,
        "add_shelves": ["books"],
        "add_tags": tags,
    }
    url = (
        "https://www.lingq.com/api/v3/en/collections/" + str(collectonID) + "/lessons/"
    )

    r = requests.post(
        url,
        headers=header,
        json=body,
    )
    print(r.json())

    level_body = {
        "ids": lesson_ids,
        "level": level,
    }

    r = requests.post(
        url,
        headers=header,
        json=level_body,
    )
    print(r.json())
    shared_body = {
        "ids": lesson_ids,
        "status": "shared",
    }

    r = requests.post(
        url,
        headers=header,
        json=shared_body,
    )
    print(r.json())
