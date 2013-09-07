// +build appengine

package handlers

import (
	"time"
)

type Registration struct {
	FirstRegistrationTime time.Time
	LastRegistrationTime  time.Time
	EnergyForPeriod       uint
	EnergyTotal           float32
}

type DailyAsJson struct {
	Date         string // Format as yyyy-mm-dd (ISO)
	Registration Registration
}

type MonthlyAsJson struct {
	Month        string // Format as yyyy-mm (ISO)
	Registration Registration
}

type YearlyAsJson struct {
	Year         string // Format as yyyy (ISO)
	Registration Registration
}

// This is stored in the datastore
type ProcessedData struct {
	Key    string         `goon:"id"`
	Parent *datastore.Key `datastore:"-" goon:"parent"`
	Json   []byte         //Json encoded array of LogPvDataForJson
}
