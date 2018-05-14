package mealplan

import (
	"encoding/base64"
	"encoding/gob"
	"math/rand"
	"os"
	"time"

	"github.com/pikans/mealplan/moira"
)

// default
const DataFile = "signups.dat"

const DateFormat = "Monday (1/2)"

// The list of duties (currently hard-coded)
var Duties = []string{"Big cook", "Little cook", "Tiny Cook", "Cleaner 1", "Cleaner 2", "Cleaner 3", "Fridge Ninja"}

// The data that is stored on disk. For "simplicity", the application just serializes and
// deserializes the entire state into / out of a single file, rather than making use of a full-blown
// database.
type Data struct {
	Days              []string
	Assignments       map[string][]moira.Username
	PlannedAttendance map[moira.Username][]bool
	VersionID         string
}

func GetDateRange() (startDate time.Time, endDate time.Time) {
	EST, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	startDate = time.Date(2018, 5, 2, 0, 0, 0, 0, EST)
	endDate = time.Date(2018, 5, 27, 0, 0, 0, 0, EST)
	return
}

func DaysIn() int {
	startDate, _ := GetDateRange()
	hoursIn := time.Now().Sub(startDate).Hours()
	return int(hoursIn / 24)
}

// Make the list of days of the current period (currently hardcoded for IAP)
func makeDayNames() []string {
	startDate, endDate := GetDateRange()
	days := []string{}
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		days = append(days, date.Format(DateFormat))
	}
	return days
}

// Make the empty state: no assignments, no planned attendance
func emptyData() *Data {
	assignments := make(map[string][]moira.Username)
	days := makeDayNames()
	for _, duty := range Duties {
		assignments[duty] = make([]moira.Username, len(days))
	}
	plannedAttendance := map[moira.Username][]bool{}
	return &Data{
		days,
		assignments,
		plannedAttendance,
		randomVersion(),
	}
}

// Read the entire data from a file
func ReadData(dataFile string) (*Data, error) {
	file, err := os.Open(dataFile)
	switch {
	case os.IsNotExist(err):
		// Doesn't exist: just use the empty state
		return emptyData(), nil
	case err != nil:
		// Some other error: return it
		return nil, err
	default:
		// Read the data out of the file
		defer file.Close()
		data := new(Data)
		dec := gob.NewDecoder(file)
		err := dec.Decode(data)
		if err != nil {
			return nil, err
		}
		data.Days = makeDayNames() // overwrite
		// If we've extended the number of days, or this is a fresh file: add blank assignments to fill
		for _, duty := range Duties {
			for len(data.Assignments[duty]) < len(data.Days) {
				data.Assignments[duty] = append(data.Assignments[duty], "")
			}
		}
		// Also extend planned attendance data
		for person := range data.PlannedAttendance {
			for len(data.PlannedAttendance[person]) < len(data.Days) {
				data.PlannedAttendance[person] = append(data.PlannedAttendance[person], false)
			}
		}
		return data, err
	}
}

// Write the entire data back to the file
func WriteData(dataFile string, data *Data) error {
	data.VersionID = randomVersion()
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	err = enc.Encode(data)
	return err
}

// Generate a random version string.
// Used to make sure saving in the admin view only goes through if no one has claimed a duty in the
// meantime (which would get overwritten).
func randomVersion() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Returns, for each day, how many people have indicated they want to come.
func (data *Data) ComputeTotalAttendance() []int {
	totals := []int{}
	for dayindex := range data.Days {
		total := 0
		for _, attends := range data.PlannedAttendance {
			if attends[dayindex] {
				total += 1
			}
		}
		totals = append(totals, total)
	}
	return totals
}
