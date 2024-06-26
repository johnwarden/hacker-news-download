download STORYID:
	mkdir -p downloads/
	DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`; time go run download-hn-item.go -s {{STORYID}} > downloads/hn-{{STORYID}}-$DATE.json
