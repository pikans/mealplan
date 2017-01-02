# pika's mealplan server

Running at <https://mealplan.pikans.org/>

## Most important files

* `data.go`: loads and saves all the state from/to disk
* `server/signup.go`: has all the logic for displaying the pages & handling user input
* `server/signup.html`: a [Go HTML template](https://golang.org/pkg/text/template/) which is used to display the main page (for both authorized and unauthorized users)

## How to deploy

Before you deploy, you'll need to clone pika's cdist repo: https://github.com/andres-erbsen/pika-yfncc-cdist.

You'll also need SSH keys for root on `pika-web.mit.edu` -- ask yfncc.

1. Make sure you're in the `server` directory
2. Build for OpenBSD: `env GOOS=openbsd GOARCH=amd64 go build`
3. Copy the resulting `server` binary to the cdist repo, at `cdist/conf/manifest/bin/openbsd/mealplan`, and copy `server.html` and `admin.html` (if changed) into `cdist/conf/manifest/html/`
4. Navigate up to `cdist/` in the cdist repo and run `./bin/cdist config -v pika-web.mit.edu`
