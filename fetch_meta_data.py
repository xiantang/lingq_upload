import argparse
import json

import requests
from bs4 import BeautifulSoup

parser = argparse.ArgumentParser(description="")
parser.add_argument("-t", "--title")


args = parser.parse_args()


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

    return {"title": title, "author": author, "description": description}


content = requests.get("https://english-e-reader.net/book/" + args.title).content
info_c = extract_info_from_html(content)
print(json.dumps(info_c, indent=4))
