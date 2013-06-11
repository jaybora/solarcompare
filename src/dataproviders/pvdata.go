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

func ToPvData(b []byte) (pvdata PvData, err error) {
	err = json.Unmarshal(b, &pvdata)
	return 
}

const KeyDateFormat = "20060102"

// Key is YYYYMMDD
type PvDataDaily map[string]uint16
