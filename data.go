package mealplan

import (
	"encoding/base64"
	"encoding/gob"
	"math/rand"
	"os"
	"time"
)

var Duties = []string{"Big cook", "Little cook", "Cleaner 1", "Cleaner 2"}

type Data struct {
	Days        []string
	Assignments map[string][]string
	VersionID   string
}

func makeDays() []string {
	EST, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	startDate := time.Date(2017, 1, 2, 0, 0, 0, 0, EST)
	endDate := time.Date(2017, 2, 12, 0, 0, 0, 0, EST)
	days := []string{}
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		days = append(days, date.Format("Monday (1/2)"))
	}
	return days
}

func emptyData() *Data {
	assignments := make(map[string][]string)
	days := makeDays()
	for _, duty := range Duties {
		assignments[duty] = make([]string, len(days))
	}
	return &Data{
		days,
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
			for len(data.Assignments[duty]) < len(data.Days) {
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
