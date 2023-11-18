import argparse
import json

import requests
from bs4 import BeautifulSoup

parser = argparse.ArgumentParser(description="")
parser.add_argument("-t", "--title")


args = parser.parse_args()


def extract_english_level_revised(html_content):
    # Parsing the HTML content
    soup = BeautifulSoup(html_content, "html.parser")

    # Searching for the English level in the vicinity of the provided snippet
    # The level seems to be within a <a> tag inside a <p> tag with class 'text-center bg-danger'
    level_tag = soup.find("dlv", class_="col-md-3 col-sm-12 hidden-sm hidden-xs")
    if level_tag and level_tag.find("a"):
        return level_tag.find("a").get_text(strip=True)

    return "English level not found"


def extract_tags_corrected(html_content):
    # Parsing the HTML content
    soup = BeautifulSoup(html_content, "html.parser")

    # The tags are found within <span class="label label-default"> inside <a> tags
    # within a <p> tag with class 'text-center'
    tags = []
    for tag in soup.find_all("span", class_="label label-default"):
        tags.append(tag.get_text(strip=True))

    return tags


def find_english_level_in_html(html_content):
    # Dictionary of English levels to search for
    levels_to_find = {
        "A1 Starter": "A1 Starter",
        "A2 Elementary": "A2 Elementary",
        "B1 Pre-Intermediate": "B1 Pre-Intermediate",
        "B1+ Intermediate": "B1+ Intermediate",
        "B2 Intermediate-Plus": "B2 Intermediate-Plus",
        "B2+ Upper-Intermediate": "B2+ Upper-Intermediate",
        "C1 Advanced": "C1 Advanced",
        "C2 Unabridged": "C2 Unabridged",
    }

    # Parsing the HTML content
    soup = BeautifulSoup(html_content, "html.parser")

    # Searching for any of the specified English levels
    for level in levels_to_find:
        if soup.find(string=lambda text: text and level in text):
            return levels_to_find[level]

    return "English level not found in document"


# Function to extract information from an HTML file
def extract_info_from_html(html_content):
    # Parsing the HTML content
    soup = BeautifulSoup(html_content, "html.parser")

    # Extracting the title
    title = (
        soup.find("title").get_text(strip=True)
        if soup.find("title")
        else "Title not found"
    )

    # Extracting the author's name from the title (assuming the format "Title - Author - Source")
    author = (
        title.split(" - ")[1] if len(title.split(" - ")) > 1 else "Author not found"
    )

    # Locating the book description
    meta_description = soup.find("meta", {"property": "og:description"})
    description = (
        meta_description["content"]
        if meta_description
        else "Book description not found"
    )
    level = find_english_level_in_html(html_content)
    tags = extract_tags_corrected(html_content)

    return {
        "title": title,
        "level": map_english_levels(level),
        "author": author,
        "description": description,
        "tags": tags,
    }


def map_english_levels(original_level):
    # Dictionary mapping the original levels to the new levels
    level_mapping = {
        "A1 Starter": "Beginner 1",
        "A2 Elementary": "Beginner 2",
        "B1 Pre-Intermediate": "Intermediate 1",
        "B1+ Intermediate": "Intermediate 2",
        "B2 Intermediate-Plus": "Advanced 1",
        "B2+ Upper-Intermediate": "Advanced 2",
        "C1 Advanced": "Advanced 1",
        "C2 Unabridged": "Advanced 2",
    }

    # Return the corresponding new level
    return level_mapping.get(original_level, "Unknown Level")


content = requests.get("https://english-e-reader.net/book/" + args.title).content
info_c = extract_info_from_html(content)
print(json.dumps(info_c, indent=4))
