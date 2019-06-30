package mealplan

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
	"github.com/pikans/mealplan/moira"
)

// default
const DataFile = "mealplan.json"

const DateFormat = "2006-01-02"

// The list of duties (currently hard-coded)
var Duties = []string{"Big Cook", "Little Cook", "Tiny Cook", "Cleaner 1", "Cleaner 2", "Cleaner 3", "Fridge Ninja", "Brunch Cook", "Brunch Cleaner"}

// The data that is stored on disk. A map of date to (map of duty to person), an end date, and a version ID in case of concurrent edits.
type Data struct {
	Assignments       map[string]map[string]moira.Username
	//removed: PlannedAttendance map[moira.Username][]bool
	EndDate           string
	VersionID         string
}

// Make the empty state: no assignments
func emptyData() *Data {
	return &Data{
		make(map[string]map[string]moira.Username),
		time.Now().AddDate(0, 1, 0).Format(DateFormat),
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
		jsonBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonBytes, &data)
		if err != nil {
			return nil, err
		}
		return data, err
	}
}

// Write the entire data back to the file
func WriteData(dataFile string, data *Data) error {
	data.VersionID = randomVersion()
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dataFile, jsonBytes, 0644)
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
