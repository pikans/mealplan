package mealplan

import (
	"encoding/base64"
	"encoding/gob"
	"math/rand"
	"os"
)

var Duties = []string{"Breakfast cook", "Breakfast cleaner", "Big cook", "Little cook", "Cleaner 1", "Cleaner 2"}
var Days = []string{
	"Monday (12/12)", "Tuesday (12/13)", "Wednesday (12/14)", "Thursday (12/15)", "Friday (12/16)", "Saturday (12/17)", "Sunday (12/18)",
	"Monday (12/19)", "Tuesday (12/20)", "Wednesday (12/21)", "Thursday (12/22)", "Friday (12/23)",
}

type Data struct {
	Assignments map[string][]string
	VersionID   string
}

func emptyData() *Data {
	assignments := make(map[string][]string)
	for _, duty := range Duties {
		assignments[duty] = make([]string, len(Days))
	}
	return &Data{
		assignments,
		randomVersion(),
	}
}

func ReadData(dataFile string) (*Data, error) {
	file, err := os.Open(dataFile)
	if err != nil {
		return emptyData(), nil
	} else {
		defer file.Close()
		data := new(Data)
		dec := gob.NewDecoder(file)
		err := dec.Decode(data)
		for _, duty := range Duties {
			for len(data.Assignments[duty]) < len(Days) {
				data.Assignments[duty] = append(data.Assignments[duty], "")
			}
		}
		return data, err
	}
}

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

func randomVersion() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}
