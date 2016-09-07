package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const dataFile = "signups.dat"

var dataLock sync.Mutex // :/

var Duties = []string{"Big cook", "Little cook", "Cleaner 1", "Cleaner 2"}
var Days = []string{"Saturday (9/10)", "Sunday (9/11)", "Monday (9/12)", "Tuesday (9/13)", "Wednesday (9/14)", "Thursday (9/15)", "Friday (9/16)"}

type Data struct {
	Assignments map[string][]string
}

func emptyData() *Data {
	assignments := make(map[string][]string)
	for _, duty := range Duties {
		assignments[duty] = make([]string, len(Days))
	}
	return &Data{
		assignments,
	}
}

type DisplayData struct {
	Days    []string
	Duties  []string
	Message string
	*Data
}

func readData() (*Data, error) {
	file, err := os.Open(dataFile)
	if err != nil {
		return emptyData(), nil
	} else {
		defer file.Close()
		data := new(Data)
		dec := gob.NewDecoder(file)
		err := dec.Decode(data)
		return data, err
	}
}

func writeData(data *Data) error {
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	err = enc.Encode(data)
	return err
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
	currentData, err := readData()
	if err != nil {
		handleErr(w, err)
		return
	}
	if ass, ok := currentData.Assignments[dutyClaimed]; ok && dayIndexClaimed < len(ass) && ass[dayIndexClaimed] == "" {
		log.Printf("%v claimed %v/%v", username, dutyClaimed, Days[dayIndexClaimed])
		ass[dayIndexClaimed] = username
	}
	err = writeData(currentData)
	if err != nil {
		handleErr(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("signup.html")
	if err != nil {
		return
	}
	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := readData()
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
