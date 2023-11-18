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
    level_tag = soup.find("p", class_="text-center bg-danger")
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
    level = extract_english_level_revised(html_content)
    tags = extract_tags_corrected(html_content)

    return {
        "title": title,
        "level": level,
        "author": author,
        "description": description,
        "tags": tags,
    }


content = requests.get("https://english-e-reader.net/book/" + args.title).content
info_c = extract_info_from_html(content)
print(json.dumps(info_c, indent=4))
