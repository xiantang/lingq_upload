import logging
import os

import requests
from dotenv import load_dotenv

logger = logging.getLogger(__name__)

load_dotenv()
key = os.getenv("APIKey")
header = {"Authorization": key, "Content-Type": "application/json"}


def generate_timestamp(lesson_id):
    logger.debug(f"Generating timestamp for lesson {lesson_id}")
    r = requests.post(
        "https://www.lingq.com/api/v3/en/lessons/" + str(lesson_id) + "/genaudio/",
        json={},
        headers=header,
    )
    if r.status_code == 200:
        logger.debug(f"Timestamp generated successfully for lesson {lesson_id}")
    else:
        logger.warning(f"Failed to generate timestamp for lesson {lesson_id} - Status: {r.status_code}")


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
    total = len(lessons["results"])
    logger.info(f"Generating timestamps for {total} lessons in collection {collectonID}")
    
    for idx, result in enumerate(lessons["results"], 1):
        lesson_id = result["id"]
        logger.debug(f"Processing lesson {idx}/{total}")
        generate_timestamp(lesson_id)
    
    logger.info(f"Timestamp generation completed for all {total} lessons")
