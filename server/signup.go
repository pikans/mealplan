package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	. "github.com/daniel-ziegler/mealplan"
)

const dataFile = "signups.dat"

var dataLock sync.Mutex // :/

type DisplayData struct {
	Days    []string
	Duties  []string
	Message string
	*Data
}

func handleErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Printf("%s\n", err)
}

func claimHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var dutyClaimed string
	var dayIndexClaimed int
	for key := range r.Form {
		splitKey := strings.Split(key, "/")
		if len(splitKey) == 3 && splitKey[0] == "claim" {
			dutyClaimed = splitKey[1]
			var err error
			dayIndexClaimed, err = strconv.Atoi(splitKey[2])
			if err != nil {
				handleErr(w, err)
				return
			}
			break
		}
	}

	username := r.Header.Get("proxy-authenticated-email")
	if username == "" {
		http.Error(w, fmt.Sprint("No username"), 401)
		return
	}
	username = strings.TrimSuffix(username, "@mit.edu")
	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := ReadData(dataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	if ass, ok := currentData.Assignments[dutyClaimed]; ok && dayIndexClaimed < len(ass) && ass[dayIndexClaimed] == "" {
		log.Printf("%v claimed %v/%v", username, dutyClaimed, Days[dayIndexClaimed])
		ass[dayIndexClaimed] = username
	}
	err = WriteData(dataFile, currentData)
	if err != nil {
		handleErr(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("signup.html")
	if err != nil {
		handleErr(w, err)
		return
	}
	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := ReadData(dataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	d := DisplayData{
		Days,
		Duties,
		"",
		currentData,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", signupHandler)
	mux.HandleFunc("/claim", claimHandler)
	return mux
}
