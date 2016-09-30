package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/daniel-ziegler/mealplan/moira"

	. "github.com/daniel-ziegler/mealplan"
)

const dataFile = "signups.dat"

var dataLock sync.Mutex // :/

type DisplayData struct {
	Days    []string
	Duties  []string
	Message string
	Unauth  bool
	*Data
}

func handleErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Printf("%s\n", err)
}

func unauthHandler(w http.ResponseWriter, r *http.Request) {
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
		true,
		currentData,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
		false,
		currentData,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

func adminAuth(w http.ResponseWriter, r *http.Request) bool {
	username := r.Header.Get("proxy-authenticated-email")
	if username == "" {
		http.Error(w, "No username", 401)
		return false
	}
	if err := moira.IsAuthorized("yfnkm", username); err != nil {
		http.Error(w, fmt.Sprintf("Not an admin: %v", username), 403)
		return false
	}
	return true
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(w, r) {
		return
	}

	t, err := template.ParseFiles("admin.html")
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
		Days:    Days,
		Duties:  Duties,
		Message: "",
		Unauth:  false,
		Data:    currentData,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func adminSaveHandler(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(w, r) {
		return
	}

	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := ReadData(dataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	oldversion := r.FormValue("oldversion")
	if got, want := oldversion, currentData.VersionID; got != want {
		http.Error(w, fmt.Sprintf("Not up to date! Got %v, wanted %v", got, want), http.StatusConflict)
		return
	}
	for _, duty := range Duties {
		for dayindex := range currentData.Assignments[duty] {
			currentData.Assignments[duty][dayindex] = r.FormValue(fmt.Sprintf("assignee/%v/%v", duty, dayindex))
		}
	}
	if err = WriteData(dataFile, currentData); err != nil {
		handleErr(w, err)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func inventoryHandler(w http.ResponseWriter, r *http.Request) {
	text, err := ioutil.ReadFile("inventory.html")
	if err != nil {
		handleErr(w, err)
		return
	}
	if _, err := w.Write(text); err != nil {
		handleErr(w, err)
		return
	}
}

func getDefaultHandler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/inventory", inventoryHandler)
	return mux
}

func getHandler() http.Handler {
	mux := getDefaultHandler()
	mux.HandleFunc("/", signupHandler)
	mux.HandleFunc("/claim", claimHandler)
	mux.HandleFunc("/admin", adminHandler)
	mux.HandleFunc("/adminSave", adminSaveHandler)
	return mux
}

func getUnauthHandler() http.Handler {
	mux := getDefaultHandler()
	mux.HandleFunc("/", unauthHandler)
	return mux
}
