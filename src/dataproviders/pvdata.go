package dataproviders

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

func (data *PvData) ToJson() (b []byte) {
	data.LatestUpdate = time.Now()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Failf("ERROR on mashalling pvdata to JSON: %s", err.Error())
	}
	return
}


const KeyDateFormat = "20060102"
// Key is YYYYMMDD
type PvDataDaily map[string]uint16

