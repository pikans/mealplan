bin_dir = /Users/j/pika/yfncc-cdist/cdist/conf/manifest/bin/openbsd
cdist = /Users/j/pika/yfncc-cdist/bin/cdist
%:
	@echo "read the README first"

build :
	go get -u all
	env GOOS=openbsd GOARCH=amd64 go build

deploy : build
	cp remind $(bin_dir)/
	$(cdist) config -v pika-web.mit.edu
