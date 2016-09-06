package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
)

var Duties = []string{"Big cook", "Little cook", "Cleaner 1", "Cleaner 2"}
var Days = []string{"Saturday (9/10)", "Sunday (9/11)", "Monday (9/12)", "Tuesday (9/13)", "Wednesday (9/14)", "Thursday (9/15)", "Friday (9/16)"}

type Data struct {
	Assignments map[string][]string
	sync.Mutex
}

var currentData Data

func emptyData() Data {
	assignments := make(map[string][]string)
	for _, duty := range Duties {
		assignments[duty] = make([]string, len(Days))
	}
	return Data{
		assignments,
		sync.Mutex{},
	}
}

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
	fmt.Fprintf(w, "hi")
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	var msg string
	switch r.Method {
	case "POST":
		msg = "post"
	case "GET":
	}
	t, err := template.ParseFiles("signup.html")
	if err != nil {
		return
	}
	currentData.Lock()
	defer currentData.Unlock()
	d := DisplayData{
		Days,
		Duties,
		msg,
		&currentData,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	currentData = emptyData()
	//http.HandleFunc("/claim", claimHandler)
	http.HandleFunc("/", signupHandler)
	http.ListenAndServe(":8080", nil)
}
