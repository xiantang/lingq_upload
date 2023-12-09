import json

from upload_book import create_collections

name = "Kidnapped"

folder = name

title = ""
detail = ""
level = ""


with open(folder + "/" + name + ".json", "r") as file:
    content = file.read()
    data = json.loads(content)
    title = data["title"]
    detail = data["detail"]
    level = data["level"]

collectionID = create_collections(
    title, detail, [], level, "https://www.eligradedreaders.com"
)
print(collectionID)
