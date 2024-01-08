for i in {1..14}; do
	curl -s -L "https://vimtricks.com/p/category/tips-and-tricks/page/$i/" | pup '.bwp-post-content > h3 > a json{}' | jq -r '.[] | [.text, .children[0].children[5].text, .href] | join(", ")' | awk -F', ' '{gsub(/\(|\)/,"",$3); print $2", "$3", "$1, $4}' | awk -F', ' '{gsub(/\(|\)/,"",$1); print $1" "$2" "$3}'
done
