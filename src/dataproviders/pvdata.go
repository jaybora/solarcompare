package dataproviders

import (
	"encoding/json"
	"time"
)

type PvData struct {
	LatestUpdate         *time.Time
	PowerAc              uint16
	PowerAcPeakAll       uint16
	PowerAcPeakAllTime   time.Time
	PowerAcPeakToday     uint16
	PowerAcPeakTodayTime time.Time
	EnergyTotal          float32
	EnergyToday          uint16
	VoltDc               float32
	AmpereAc             float32
	State                string
}

func (data *PvData) ToJson() (b []byte) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Failf("ERROR on mashalling pvdata to JSON: %s", err.Error())
	}
	return
}

func FromJson(b []byte) PvData {
	p := PvData{}
	err := json.Unmarshal(b, &p)
	if err != nil {
		log.Failf("ERROR on unmashalling pvdata from JSON: %s", err.Error())
	}
	return p
}

const KeyDateFormat = "20060102"

// Key is YYYYMMDD
type PvDataDaily map[string]uint16
