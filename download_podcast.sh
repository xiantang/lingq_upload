urls="$(curl -s https://teacherluke.co.uk/archive-of-episodes-1-149/ | pup '#post-1043 > div > p > strong > a attr{href}' | tac)"

for url in $urls; do
	# title=$(
	# 	curl -s "$url" | pup 'header > h1 text{}'
	# )
	extracted_part="${url%/}"              # 去除 URL 末尾的斜线（如果存在）
	extracted_part="${extracted_part##*/}" # 提取最后一个斜线后的内容
	echo "$extracted_part"

	curl -s "$url" | pup "a attr{href}" | grep "https://open.acast.com" | xargs wget -O "luke/${extracted_part}.mp3"
done
