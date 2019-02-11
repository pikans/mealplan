bin_dir = /Users/j/pika/yfncc-cdist/cdist/conf/manifest/bin/openbsd
html_dir = /Users/j/pika/yfncc-cdist/cdist/conf/manifest/html
cdist = /Users/j/pika/yfncc-cdist/bin/cdist

warn:
	@echo "read the README.md first."

build :
	go get -u github.com/pikans/mealplan github.com/pikans/mealplan/moira golang.org/x/crypto/acme/autocert
	env GOOS=openbsd GOARCH=amd64 go build

deploy : build
	cp signup.html admin.html $(html_dir)/
	cp server $(bin_dir)/
	$(cdist) config -v pika-web.mit.edu
