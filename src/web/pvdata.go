package web

import (
	"time"
	"encoding/json"
)

type PvData struct {
	LatestUpdate time.Time
	PowerAc      uint16
	EnergyTotal  float32
	EnergyToday  uint16
	VoltDc       float32
	AmpereAc     float32}

func (data *PvData) ToJson() (b []byte, err error) {
	data.LatestUpdate = time.Now()
	b, err = json.MarshalIndent(data, "", "  ")
	return
}




