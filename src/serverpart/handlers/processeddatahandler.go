// +build appengine

package handlers

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"goon"
	"net/http"
	"plantdata"
	"time"
)

type Registration struct {
	FirstRegistrationTime time.Time
	LastRegistrationTime  time.Time
	EnergyForPeriod       uint
	EnergyTotal           float32
}

// This is for json output.
// A map where key is one of
//  -  date as yyyymmdd partioned by month
//  -  month as yyyymm partioned by year
//  -  year as yyyy all years in one file (so far)
// So a single entity in datastore will hold
// a map with all registrations for that partioned period, with a registration
// per day/month/year (as a json string)
type RegistrationsAsJson struct {
	Registrations map[string]Registration
}

// This is stored in the datastore
type ProcessedData struct {
	Key    string         `goon:"id"`
	Parent *datastore.Key `datastore:"-" goon:"parent"`
	Json   []byte         //Json encoded array of LogPvDataForJson
}

// An enum thingie for golang
type ProssesingType int

const (
	DAILY ProcessiongType = iota
	MONTHLY
	YEARLY
)

type DataProcessor struct {
	Processor     ProssesingType
	Plant         *plantdata.Plant
	TimeKey       *string // yyyymmdd for daily, yyyymm for monthly, yyyy for yearly
	Request       *http.Request
	context       *appengine.Context
	registrations *RegistrationsAsJson
}

// Process pvdata for the time and period given by the base type DataProcessor
func (dp *DataProcessor) ProcessPvData() Error {
	// Get the entity with data for our partittion
	err := dp.loadRegistrations()
	if err != nil {
		return err
	}

	// Then we need to get some data from the datastore
	// For daily we will use the LogPvDataDaily
	pvdatalogs, err := GetLogPvData(dp.Plant, dp.Request, *dp.TimeKey)
	if err != nil {
		return err
	}

	// Add new registration to map
	firstReg := pvdatalogs[0]
	lastReg := pvdatalogs[len(pvdatalogs)-1]

	dp.registrations.Registrations[*dp.TimeKey] =
		Registration{firstReg.LogTime,
			lastReg.LogTime,
			lastReg.PvData.EnergyToday,
			lastReg.PvData.EnergyTotal}

	return nil

}

// Load up exsisting registrations from datastore to map
func (dp *DataProcessor) loadRegistrations() Error {
	// Get entity. Dont need to do anything about an error
	entity, _ := getProcessedDataEntity(dp.Request, dp.Plant, dp.getEntityKey())

	// Build exsisting data from json to map
	var registrationsAsJson = RegistrationsAsJson{}
	if len(entity.Json) > 0 {
		err = json.Unmarshal(entity.Json, &registrationsMap.Registrations)
		if err != nil {
			c.Errorf("Could not unmarshal json ProcessedData from datastore due to %s for plant %s", err.Error(), dp.Plant.PlantKey)
			return err
		}

	}
	dp.registrations = registrationsAsJson
	return nil
}

// Persist from map to datastore
func (dp *DataProcessor) persistRegistrations() Error {
	// Get entity. Dont need to do anything about an error
	entity, _ := getProcessedDataEntity(dp.Request, dp.Plant, dp.getEntityKey())

	// Build exsisting data from json to map
	var registrationsAsJson = RegistrationsAsJson{}
	if len(entity.Json) > 0 {
		err = json.Unmarshal(entity.Json, &registrationsMap.Registrations)
		if err != nil {
			c.Errorf("Could not unmarshal json ProcessedData from datastore due to %s for plant %s", err.Error(), dp.Plant.PlantKey)
			return err
		}

	}
	dp.registrations = registrationsAsJson
	return nil
}

func (dp *DataProcessor) getEntityKey() string {
	k := []byte(dp.TimeKey)
	switch dp.Processor {
	case DAILY:
		return k[0:6]
	case MONTHLY:
		return k[0:4]
	case YEARLY:
		return "ALL"
	}
	return nil
}

func getProcessedDataEntity(r *http.Request, plant *plantdata.Plant, key *string) (pd ProcessedData, err Error) {
	g := goon.NewGoon(r)
	c := appengine.NewContext(r)

	pd.Key = *key

	plantkey, err := g.GetStructKey(plant)
	if err != nil {
		c.Infof("Got an error when getting ProcessedDataEntity from datastore, %s", err.Error())
	}
	pd.Parent = plantkey
	err = g.Get(&pd)
	if err != nil {
		c.Infof("Got an error when getting ProcessedDataEntity from datastore, %s", err.Error())
	}
	// If none was found a new ProcessedData is then returned
	return
}

/* Princip for log monthly
 * This will go log all day productions for a given month.
 * If run without parameters, default from app engine cron
 * then update for the current month.
 * Go through all plants
 * - Get existing monthlogrecord from datastore. Create if dont exists
 * - - Add logrecord to json for days missing before now. Calculate production by total production value minus previous days value
 * - Close logrecord and switch plant
 */

// func logMonthly(w http.ResponseWriter, r *http.Request) {
// 	c := appengine.NewContext(r)
// 	g := goon.NewGoon(r)
// }
