package plantdata

import (
	"dataproviders"
	"encoding/json"
)

type Inverter struct {
	Vendor   string
	Model    string
	Capacity int16
}

type Panels struct {
	Vendor   string
	Model    string
	Capacity int16
	Pieces   int16
}

type Plant struct {
	PlantKey     string `goon:"id"`
	User         string `json:"-"`
	Name         string
	Latitide     string
	Longitude    string
	Picture      []byte
	Panels       Panels
	Inverter     Inverter
	InitiateData dataproviders.InitiateData
	//PvData       dataproviders.PvData       `json:"-"` //Live data
	// The dataproviders implementation
	DataProvider int
}

func (data *Plant) ToJson() (b []byte, err error) {
	b, err = json.MarshalIndent(data, "", "  ")
	return
}

func ToPlant(jsonBytes *[]byte) (plantdata Plant, err error) {
	err = json.Unmarshal(*jsonBytes, &plantdata)
	return
}
