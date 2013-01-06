package web

import (
	"encoding/json"
	"../dataproviders"
	"../web"
)

type PlantData struct {
	PlantKey     string
	Name         string
	Latitide     string
	Longitude    string
	Picture      []byte
	PvData       web.PvData
	DataProvider dataprovider.DataProvider
}

func (data *PlantData) ToJson() (b []byte, err error) {
	b, err = json.MarshalIndent(data, "", "  ")
	return
}
