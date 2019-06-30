package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"
	"github.com/pikans/mealplan/moira"
	. "github.com/pikans/mealplan"
)

// Use a mutex to prevent concurrent access to the data file.
// It's a bit unfortunate to control access to a file system resource using an in-memory mutex in
// the server, but it's simple.
var dataLock sync.Mutex

// The data type which will be passed to the HTML template (signup.html).
type DisplayData struct {
	Duties      []string
	Authorized  bool
	Username    moira.Username
	DayNames    map[string]string
	Weeks       [][]string
	Assignments map[string]map[string]moira.Username
	VersionID   string
}

func makeWeeksAndDayNames(endDate string) ([][]string, map[string]string) {
	weeks := [][]string{}
	dayNames := map[string]string{}

	today := time.Now()
	todayOffset := today.Weekday() - time.Monday
	if todayOffset < 0 {
		todayOffset += 7
	}
	actualStart := today.AddDate(0, 0, -int(todayOffset))

	end, _ := time.Parse(DateFormat, endDate)
	endOffset := time.Sunday - end.Weekday()
	if endOffset < 0 {
		endOffset += 7
	}
	actualEnd := end.AddDate(0, 0, int(endOffset))

	for day := actualStart; day.Year() < actualEnd.Year() || (day.Year() == actualEnd.Year() && day.YearDay() <= actualEnd.YearDay()); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Monday {
			weeks = append(weeks, []string{})
		}
		dayString := day.Format(DateFormat)
		weeks[len(weeks)-1] = append(weeks[len(weeks)-1], dayString)
		dayNames[dayString] = day.Format("Monday (1/2)")
	}
	return weeks, dayNames
}


func handleErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Printf("%s\n", err)
}

// This handler runs for unauthorized users (no certs / not on pika-food).
// It displays all the claimed duties, but doesn't display
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
	currentData, err := ReadData(DataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	weeks, dayNames := makeWeeksAndDayNames(currentData.EndDate)
	d := DisplayData{
		Duties:      Duties,
		Authorized:  false,
		Username:    "",
		DayNames:    dayNames,
		Weeks:       weeks,
		Assignments: currentData.Assignments,
		VersionID:   currentData.VersionID,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// This handler displays the main signup page for authorized users (certs & on pika-food).
// It displays buttons and checkboxes to enable the user to claim duties.
func signupHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("signup.html")
	if err != nil {
		handleErr(w, err)
		return
	}
	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := ReadData(DataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	username := getAuthedUsername(r)
	if username == "" {
		http.Error(w, "No username", http.StatusUnauthorized)
		return
	}
	log.Printf("displaying for user %v", username)
	for _, duty := range Duties {
		// If duties contain slashes, the logic in claimHandler will break, because the button IDs use
		// slashes as separators (see signup.html).
		if strings.Contains(duty, "/") {
			panic("duties can't contain slashes")
		}
	}
	weeks, dayNames := makeWeeksAndDayNames(currentData.EndDate)
	d := DisplayData{
		Duties:      Duties,
		Authorized:  true,
		Username:    username,
		DayNames:    dayNames,
		Weeks:       weeks,
		Assignments: currentData.Assignments,
		VersionID:   currentData.VersionID,
	}
	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getAuthedUsername(r *http.Request) moira.Username {
	email := moira.Email(r.Header.Get("proxy-authenticated-email"))
	return moira.UsernameFromEmail(email)
}

func transact(f func(*Data) error) error {
	dataLock.Lock()
	defer dataLock.Unlock()

	currentData, err := ReadData(DataFile)
	if err != nil {
		return err
	}

	if err := f(currentData); err != nil {
		return err
	}

	return WriteData(DataFile, currentData)
}

// This handler runs when users submit the form (by clicking Save or a duty-claiming button).
// It updates the on-disk data correspondingly, and then sends users back to the main page.
func claimHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	r.ParseForm()

	username := getAuthedUsername(r)
	if username == "" {
		http.Error(w, "No username", http.StatusUnauthorized)
	}

	// Find whether a duty was claimed, and if so, which one
	for key := range r.Form {
		splitKey := strings.Split(key, "/")
		if len(splitKey) == 3 && splitKey[0] == "claim" {
			duty := splitKey[1]
			day := splitKey[2]
			err := transact(func(currentData *Data) error {
				dayAssignments, ok := currentData.Assignments[day]
				if !ok {
					dayAssignments = make(map[string]moira.Username)
					currentData.Assignments[day] = dayAssignments
				}
				assignee, ok := dayAssignments[duty]
				if ok && assignee != "" {
					return errors.New("somebody else got this one already.")
				}
				dayAssignments[duty] = username
				return nil
			})
			if err != nil {
				break
			}
			log.Printf("%v claimed %v/%v", username, duty, day)
			break
		}
		if len(splitKey) == 3 && splitKey[0] == "abandon" {
			duty := splitKey[1]
			day := splitKey[2]
			err := transact(func(currentData *Data) error {
				dayAssignments, ok := currentData.Assignments[day]
				if !ok {
					return errors.New("not yours, no need to abandon it.")
				}
				assignee, ok := dayAssignments[duty]
				if !ok || assignee != username {
					return errors.New("not yours, no need to abandon it.")
				}
				dayAssignments[duty] = ""
				return nil
			})
			if err != nil {
				break
			}

			log.Printf("%v abandoned %v/%v", username, duty, day)

			err = smtp.SendMail(
				"outgoing.mit.edu:smtp",
				nil,
				"yfnkm@mit.edu",
				[]string{"yfnkm@mit.edu", fmt.Sprint(username.Email())},
				[]byte(fmt.Sprintf(`From: "pika kitchen website" <yfnkm@mit.edu>
To: yfnkm@mit.edu
Cc: %s
Subject: %s unclaimed %v/%v -- eom

`, username.Email(), username, duty, day)))
			if err != nil {
				log.Printf("%v", err)
			}

			break
		}
	}

	// Display the main page again
	http.Redirect(w, r, "/", http.StatusFound)
}

// Authorizes the user as admin (must be on yfnkm or yfncc); aborts the request
// with 403 Forbidden if not.  Returns whether authorization succeeded.
func adminAuth(w http.ResponseWriter, r *http.Request) bool {
	username := getAuthedUsername(r)
	if username == "" {
		http.Error(w, "No username", http.StatusUnauthorized)
		return false
	}
	if err := moira.IsAuthorized("yfnkm", username); err == nil {
		return true
	}
	if err := moira.IsAuthorized("yfncc", username); err == nil {
		return true
	}
	http.Error(w, fmt.Sprintf("Not an admin (yfnkm or yfncc): %v", username), http.StatusForbidden)
	return false
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
	currentData, err := ReadData(DataFile)
	if err != nil {
		handleErr(w, err)
		return
	}
	weeks, dayNames := makeWeeksAndDayNames(currentData.EndDate)
	d := DisplayData{
		Duties:      Duties,
		Authorized:  true,
		Username:    "",
		DayNames:    dayNames,
		Weeks:       weeks,
		Assignments: currentData.Assignments,
		VersionID:   currentData.VersionID, // Store the version in a hidden field
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
	currentData, err := ReadData(DataFile)
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
	_, dayNames := makeWeeksAndDayNames(currentData.EndDate)
	for day, _ := range dayNames {
		dayAssignments, ok := currentData.Assignments[day]
		if !ok {
			dayAssignments = make(map[string]moira.Username)
			currentData.Assignments[day] = dayAssignments
		}
		for _, duty := range Duties {
			if values, ok := r.Form[fmt.Sprintf("assignee/%v/%v", duty, day)]; ok && len(values) != 0 {
				dayAssignments[duty] = moira.Username(values[0])
			}
		}
	}
	if err = WriteData(DataFile, currentData); err != nil {
		handleErr(w, err)
		return
	}

	// Display the admin interface again
	http.Redirect(w, r, "/admin", http.StatusFound)
}

/* stats page is disabled for now, you'll need to do some refactoring if you want to bring it back 

type Signup struct {
	Date, Duty string
}

type PersonStats struct {
	Signups  []Signup
	Username moira.Username
}

type BySignupCount []PersonStats

func (s BySignupCount) Len() int {
	return len(s)
}
func (s BySignupCount) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s BySignupCount) Less(i, j int) bool {
	return len(s[i].Signups) < len(s[j].Signups)
}

type StatsData struct {
	People []PersonStats
	Since  time.Time
}

func adminStatsHandler(w http.ResponseWriter, r *http.Request) {
	if !adminAuth(w, r) {
		return
	}

	t, err := template.ParseFiles("stats.html")
	if err != nil {
		handleErr(w, err)
		return
	}
	dataLock.Lock()
	defer dataLock.Unlock()
	currentData, err := ReadData(DataFile)
	if err != nil {
		handleErr(w, err)
		return
	}

	authorize := r.Header.Get("proxy-authorized-list")
	users, err := moira.GetMoiraNFSGroupMembers(authorize)
	if err != nil {
		handleErr(w, err)
		return
	}

	stats := map[moira.Username]PersonStats{}
	for _, u := range users {
		stats[u] = PersonStats{Signups: []Signup{}, Username: u}
	}

	mealplanStartDate := mealplanStartDate()
	dbStartDate, _ := GetDateRange()
	for dayindex, dayname := range currentData.Days {
		date := dbStartDate.AddDate(0, 0, dayindex)
		if date.Equal(mealplanStartDate) || date.After(mealplanStartDate) {
			for _, duty := range Duties {
				if dayindex < len(currentData.Assignments[duty]) {
					u := currentData.Assignments[duty][dayindex]
					if u != "" && u != "_" {
						stats[u] = PersonStats{append(stats[u].Signups, Signup{dayname, duty}), u}
					}
				}
			}
		}
	}

	d := StatsData{People: []PersonStats{}, Since: mealplanStartDate}
	for _, s := range stats {
		// if len(s.Signups) != 0 {
		d.People = append(d.People, s)
	}
	sort.Sort(BySignupCount(d.People))

	err = t.Execute(w, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
*/

// This is the overall handler which decides, for authorized users, which page to display.
func getHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", signupHandler)
	mux.HandleFunc("/claim", claimHandler)
	mux.HandleFunc("/admin", adminHandler)
	mux.HandleFunc("/adminSave", adminSaveHandler)
//	mux.HandleFunc("/stats", adminStatsHandler)
	return mux
}

// This is the overall handler for unauthorized users. It always displays the unauthorized
// interface.
func getUnauthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", unauthHandler)
	return mux
}
