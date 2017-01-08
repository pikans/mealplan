package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/daniel-ziegler/mealplan/moira"

	. "github.com/daniel-ziegler/mealplan"
)

const dataFile = "signups.dat"

// Use a mutex to prevent concurrent access to the data file.
// It's a bit unfortunate to control access to a file system resource using an in-memory mutex in
// the server, but it's simple.
var dataLock sync.Mutex

// The data type which will be passed to the HTML template (signup.html).
type DisplayData struct {
	Duties                       []string
	Authorized                   bool
	Username                     string
	DayNames                     []string
	Weeks                        [][]int
	CurrentUserPlannedAttendance []bool
	TotalAttendance              []int
	Assignments                  map[string][]string
	VersionID                    string
}

func makeWeeks(nrDays int) [][]int {
	weeks := [][]int{}
	for i := 0; i < nrDays; i++ {
		if i%7 == 0 {
			weeks = append(weeks, []int{})
		}
		weeks[len(weeks)-1] = append(weeks[len(weeks)-1], i)
	}
	return weeks
}

func handleErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Printf("%s\n", err)
}

// This handler runs for unauthorized users (no certs / not on pika-food).
// It displays all the claimed duties and the indicated attendance counts, but doesn't display
// buttons or checkboxes for the users to make any changes. (This is taken care of in signup.html,
// which checks .Authorized on the data to check whether the user is authorized or not.)
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
		Duties:     Duties,
		Authorized: false,
		Username:   "",
		DayNames:   currentData.DayNames,
		Weeks:      makeWeeks(len(currentData.DayNames)),
		CurrentUserPlannedAttendance: nil,
		TotalAttendance:              currentData.ComputeTotalAttendance(),
		Assignments:                  currentData.Assignments,
		VersionID:                    currentData.VersionID,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// This handler displays the main signup page for authorized users (certs & on pika-food).
// It displays buttons and checkboxes to enable the user to claim duties and indicate the days they
// plan on attending dinner.
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
	username := getTrimmedUsername(r)
	if username == "" {
		http.Error(w, "No username", http.StatusUnauthorized)
		return
	}
	log.Printf("displaying for user %v", username)
	plan, ok := currentData.PlannedAttendance[username]
	if !ok {
		plan = make([]bool, len(currentData.DayNames))
	}
	for _, duty := range Duties {
		// If duties contain slashes, the logic in claimHandler will break, because the button IDs use
		// slashes as separators (see signup.html).
		if strings.Contains(duty, "/") {
			panic("duties can't contain slashes")
		}
	}
	d := DisplayData{
		Duties:     Duties,
		Authorized: true,
		Username:   username,
		DayNames:   currentData.DayNames,
		Weeks:      makeWeeks(len(currentData.DayNames)),
		CurrentUserPlannedAttendance: plan,
		TotalAttendance:              currentData.ComputeTotalAttendance(),
		Assignments:                  currentData.Assignments,
		VersionID:                    currentData.VersionID,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Returns either a Kerberos username (with @mit.edu trimmed off) or a whole email
func getTrimmedUsername(r *http.Request) string {
	username := r.Header.Get("proxy-authenticated-email")
	return strings.TrimSuffix(username, "@mit.edu")
}

// This handler runs when users submit the form (by clicking Save or a duty-claiming button).
// It updates the on-disk data correspondingly, and then sends users back to the main page.
func claimHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()

		// Find whether a duty was claimed, and if so, which one
		var dutyClaimed string
		var dayIndexClaimed int
		var claimingSomething bool
		for key := range r.Form {
			splitKey := strings.Split(key, "/")
			if len(splitKey) == 3 && splitKey[0] == "claim" {
				claimingSomething = true
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

		username := getTrimmedUsername(r)
		if username == "" {
			http.Error(w, "No username", http.StatusUnauthorized)
		}

		dataLock.Lock()
		defer dataLock.Unlock()
		currentData, err := ReadData(dataFile)
		if err != nil {
			handleErr(w, err)
			return
		}

		if claimingSomething {
			// Claim the duty
			if ass, ok := currentData.Assignments[dutyClaimed]; ok && dayIndexClaimed < len(ass) && ass[dayIndexClaimed] == "" {
				log.Printf("%v claimed %v/%v", username, dutyClaimed, currentData.DayNames[dayIndexClaimed])
				ass[dayIndexClaimed] = username
			}
		}
		// Also update planned attendance
		plannedAttendance := make([]bool, len(currentData.DayNames))
		for dayindex := range currentData.DayNames {
			vals := r.Form[fmt.Sprintf("attend/%d", dayindex)]
			willAttend := len(vals) == 1 && vals[0] == "true"
			plannedAttendance[dayindex] = willAttend
		}
		currentData.PlannedAttendance[username] = plannedAttendance

		err = WriteData(dataFile, currentData)
		if err != nil {
			handleErr(w, err)
			return
		}
	}
	// Display the main page again
	http.Redirect(w, r, "/", http.StatusFound)
}

// Authorizes the user as admin (must be on yfnkm); aborts the request with 403 Forbidden if not.
// Returns whether authorization succeeded.
func adminAuth(w http.ResponseWriter, r *http.Request) bool {
	username := r.Header.Get("proxy-authenticated-email")
	if username == "" {
		http.Error(w, "No username", http.StatusUnauthorized)
		return false
	}
	if err := moira.IsAuthorized("yfnkm", username); err != nil {
		http.Error(w, fmt.Sprintf("Not an admin: %v", username), http.StatusForbidden)
		return false
	}
	return true
}

// This handler displays the secret admin interface, which displays a bunch of textboxes rather than
// merely claim buttons, allowing yfnkm to make arbitrary changes to the claimed duties.
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
		Duties:     Duties,
		Authorized: true,
		Username:   "",
		DayNames:   currentData.DayNames,
		Weeks:      makeWeeks(len(currentData.DayNames)),
		CurrentUserPlannedAttendance: nil,
		Assignments:                  currentData.Assignments,
		VersionID:                    currentData.VersionID, // Store the version in a hidden field
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// This handler runs when the admin hits "Save" on the admin interface.
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
	// Compare the current version string with the version string stored in a hidden field when the
	// page was originally displayed. If there has been a change in the meantime, abort -- this could
	// lead to overwriting duties that other people claimed (since the entire state gets overwritten
	// with the contents of the textboxes on the page). This has saved my ass at least once!
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

	// Display the admin interface again
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// This is the overall handler which decides, for authorized users, which page to display.
func getHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", signupHandler)
	mux.HandleFunc("/claim", claimHandler)
	mux.HandleFunc("/admin", adminHandler)
	mux.HandleFunc("/adminSave", adminSaveHandler)
	return mux
}

// This is the overall handler for unauthorized users. It always displays the unauthorized
// interface.
func getUnauthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", unauthHandler)
	return mux
}
