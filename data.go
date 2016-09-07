package mealplan

import (
	"encoding/gob"
	"os"
)

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

func ReadData(dataFile string) (*Data, error) {
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

func WriteData(dataFile string, data *Data) error {
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	err = enc.Encode(data)
	return err
}
