# pika's mealplan server

Running at <https://mealplan.pikans.org/>

## Most important files

* `data.go`: loads and saves all the state from/to disk
* `server/signup.go`: has all the logic for displaying the pages & handling user input
* `server/signup.html`: a [Go HTML template](https://golang.org/pkg/text/template/) which is used to display the main page (for both authorized and unauthorized users)

## How to deploy

# Somewhat less manual way

Before you deploy, you'll need to clone pika's cdist repo: <https://wiki.pikans.org/computers/git#yfncc-cdist>. Note down the path you clone this to. Navigate to the directory for the thing you'd like to change (either server or remind, probably), then edit the makefile variables at the top to match the prefixes to the path to yfncc-cdist you noted down earlier (i.e., replace `/Users/j/pika/yfncc-dist` with the full path to your copy of yfncc-cdist).

Now you can build the project with `make build`, and have cdist put it on the mealplan server with `make deploy`.

Now follow the manual steps below from step 5.

# Somewhat more manual way

You'll also need SSH keys for root on `pika-web.mit.edu` -- ask yfncc.

1. Make sure you're in the `server` directory
2. Build for OpenBSD: `env GOOS=openbsd GOARCH=amd64 go build`
3. Copy the resulting `server` binary to the cdist repo, at `cdist/conf/manifest/bin/openbsd/mealplan`, and copy `server.html` and `admin.html` (if changed) into `cdist/conf/manifest/html/`
4. Navigate up to `yfncc-cdist/` in the cdist repo and run `./bin/cdist config -v pika-web.mit.edu`
5. (If the binary changed) SSH into `pika-web.mit.edu` and restart the server:
  1. `su mealplan` to change into user `mealplan`
  2. `tmux attach` to attach to an existing `tmux` session, or just `tmux` to start a new one
  3. Kill the existing `mealplanserver` process, either with Ctrl+C if it's in the console, or finding the process and stopping it.
  4. Start it anew, with `./run.sh` or `./run-console.sh`, depending on whether you want to get live console output in the `tmux` session (*only* do this if you're running in `tmux`, or else it will die once your SSH connection dies!)

## NOTE
	Make sure when changing anything in this repo to commit, push, and
    execute `go get github.com/pikans/mealplan`; otherwise, parts of
    this development that refer to other files in the repo by use of a
    go remote import from the github repo will not use the updated
    bits!
