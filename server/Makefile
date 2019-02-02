bin_dir = /Users/j/pika/yfncc-cdist/cdist/conf/manifest/bin/openbsd
cdist = /Users/j/pika/yfncc-cdist/bin/cdist
server :
	env GOOS=openbsd GOARCH=amd64 go build
deploy : remind
	cp server $(bin_dir)/
	$(cdist) config -v pika-web.mit.edu
