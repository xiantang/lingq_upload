import logging
import os

import requests
from dotenv import load_dotenv

from generate_timestamp import get_lessons

logger = logging.getLogger(__name__)

load_dotenv()
key = os.getenv("APIKey")


header = {"Authorization": key, "Content-Type": "application/json"}


def update_metadata(collectonID, tags, level):
    logger.debug(f"Updating metadata for collection {collectonID} - Level: {level}, Tags: {tags}")
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
    if r.status_code == 200:
        logger.info(f"Updated tags and shelves for {len(lesson_ids)} lessons")
    else:
        logger.error(f"Failed to update tags/shelves - Status: {r.status_code}")

    level_body = {
        "ids": lesson_ids,
        "level": level,
    }

    r = requests.post(
        url,
        headers=header,
        json=level_body,
    )
    if r.status_code == 200:
        logger.info(f"Updated level to {level} for {len(lesson_ids)} lessons")
    else:
        logger.error(f"Failed to update level - Status: {r.status_code}")
    shared_body = {
        "ids": lesson_ids,
        "status": "shared",
    }

    r = requests.post(
        url,
        headers=header,
        json=shared_body,
    )
    if r.status_code == 200:
        logger.info(f"Set status to 'shared' for {len(lesson_ids)} lessons")
    else:
        logger.error(f"Failed to update status - Status: {r.status_code}")
