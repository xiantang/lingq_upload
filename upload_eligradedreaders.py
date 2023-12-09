import json

from upload_book import create_collections

name = "Kidnapped"

folder = name


with open(folder + "/" + name + ".json", "r") as file:
    content = file.read()
    data = json.loads(content)
    print(data)
