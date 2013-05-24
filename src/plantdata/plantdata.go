package plantdata

import (
	"dataproviders"
	"encoding/json"
)

type InverterData struct {
	InverterVendor   string
	InverterModel    string
	InverterCapacity uint16
}

type CellData struct {
	CellVendor   string
	CellModel    string
	CellCapatity uint16
}

type PlantData struct {
	PlantKey     string
	Name         string
	Latitide     string
	Longitude    string
	Picture      []byte
	CellData     CellData
	InverterData InverterData
	InitiateData dataproviders.InitiateData 
	PvData       dataproviders.PvData       `json:"-"` //Live data 
	// The dataproviders implementation
	DataProvider int
}

func (data *PlantData) ToJson() (b []byte, err error) {
	b, err = json.MarshalIndent(data, "", "  ")
	return
}
