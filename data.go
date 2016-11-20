package mealplan

import (
	"encoding/base64"
	"encoding/gob"
	"math/rand"
	"os"
)

var Duties = []string{"Big cook", "Little cook", "Cleaner 1", "Cleaner 2"}
var Days = []string{"Wednesday (11/23)", "Thursday (11/24)", "Friday (11/25)", "Saturday (11/26)", "Sunday (11/27)"}

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
