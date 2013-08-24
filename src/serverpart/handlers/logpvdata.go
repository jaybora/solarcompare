// +build appengine

package handlers

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"dataproviders"
	"encoding/json"
	"fmt"
	"goon"
	"net/http"
	"plantdata"
	"time"
	"web"
)

const keyDateFormat = "20060102"
const keyMonthFormat = "200601"

// This is for the marshalling to json
type LogPvDataForJson struct {
	LogTime         time.Time
	PvData          dataproviders.PvData
	EnergyTotalDiff float32
	EnergyTodayDiff uint16
}

// This is stored in the datastore
type LogPvDataDaily struct {
	LogKey string         `goon:"id"`
	Parent *datastore.Key `datastore:"-" goon:"parent"`
	Logs   []byte         //Json encoded array of LogPvDataForJson
}

// This handler is called by app engine cron to regular
// log data to datastore

func LogPvDataHandler(w http.ResponseWriter, r *http.Request) {
	// This loghandler looks at the urlparam "logtype" for
	// what action to take:
	// logtype=daily: Log pvdata as
	// it is right now. One log per plant per day per registration type
	// Stores a json string with the data in the form of an array, where
	// each element is time, pvdata.
	// Key for datastore is composite of plantkey/yyyyMMdd
	// logtype=monthly: Log

	c := appengine.NewContext(r)
	if r.URL.Query().Get("logtype") == "daily" {
		logDaily(w, r)
	} else {
		keypos := 2
		if !appengine.IsDevAppServer() {
			keypos += 2
		}

		plantkey := web.PlantKey(r.URL.String(), keypos)
		if plantkey != "" {
			p, err := Plant(r, &plantkey)

			if err != nil {
				c.Infof("Could not find plat due to %s", err.Error())
				http.Error(w, fmt.Sprintf(err.Error()),
					http.StatusInternalServerError)
				return
			}

			date := r.URL.Query().Get("date")

			logdatadaily := getLogDaily(p, r, &date)
			w.Header().Add("Content-type", "application/json")
			w.Write(logdatadaily.Logs)
			return

		} else {

			http.Error(w, "Currently url parameter can only be ?logtype=daily",
				http.StatusBadRequest)
		}
	}

}

// Get the plants we should log for
func logDailyGetPlants(w http.ResponseWriter, r *http.Request) *datastore.Query {
	return datastore.NewQuery("Plant").
		Order("PlantKey").
		Limit(100)
}

func logDaily(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	q := logDailyGetPlants(w, r)
	logtime := time.Now()
	okcounter := 0
	for i := g.Run(q); ; {
		p := plantdata.Plant{}
		_, err := i.Next(&p)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Infof("Could not list plants for dailylog from datastore due to %s", err.Error())
			http.Error(w, fmt.Sprintf(err.Error()),
				http.StatusInternalServerError)
			return
		}

		// OK p is now a plant.
		// Get daily log from datastore
		c.Debugf("Getting PvData log for plantkey %s", p.PlantKey)

		logDaily := getLogDaily(&p, r, nil)
		c.Debugf("Getting PvData log for plantkey %s", p.PlantKey)

		var exsistingLogs []LogPvDataForJson
		if len(logDaily.Logs) > 0 {
			err = json.Unmarshal(logDaily.Logs, &exsistingLogs)
			if err != nil {
				c.Errorf("Could not unmarshal json PvDataLog from datastore due to %s for plant %s", err.Error(), p.PlantKey)
				continue
			}
		}

		// Get pvdata to add to json log
		c.Debugf("Getting PvData log for plantkey %s", p.PlantKey)
		pvdata, err := getPvData(&p.PlantKey, r)
		if err != nil {
			c.Errorf("%s", err.Error())
			continue
		}
		//Logtime to keep all logs for this jobrun syncronizes
		newLog := LogPvDataForJson{logtime, pvdata}
		newLogs := append(exsistingLogs, newLog)

		// Make it a json string again
		b, err := json.MarshalIndent(&newLogs, "", "  ")
		if err != nil {
			c.Errorf("ERROR on mashalling logpvdata to JSON: %s", err.Error())
		}
		logDaily.Logs = b

		//Put it in store
		putLogDaily(&p, time.Now(), r, logDaily)
		okcounter++

	}
	fmt.Fprintf(w, "Done. Logged PvData for %d plants", okcounter)

}

// Get PvData for log.
// First check for recent data in memcache.
// Otherwise request a startup on the dpu
// Wait a title an look in the memcache again
func getPvData(plantkey *string, r *http.Request) (pvdata dataproviders.PvData, err error) {
	c := appengine.NewContext(r)
	for tries := 0; tries < 5; tries++ {
		i, errlocal := memcache.Get(c, keyPvData(plantkey))
		if errlocal != nil {
			c.Debugf("Could not get PvData from memcache for pvdatalog for %s due to %s",
				*plantkey, errlocal.Error())
			startDpUnit(r, plantkey)
			time.Sleep(10 * time.Second)
		} else if checkForRestart(c, i.Value) {
			startDpUnit(r, plantkey)
			time.Sleep(10 * time.Second)
		} else {
			pvdata, _ = dataproviders.ToPvData(i.Value)
			c.Debugf("Got a useful PvData to log. PvData is %s", pvdata)
			err = nil
			return
		}
	}
	err = fmt.Errorf("Could not get data for plant %s for pvdatalog. Giving up.", *plantkey)
	return
}

func getLogDaily(plant *plantdata.Plant, r *http.Request, date *string) *LogPvDataDaily {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)

	l := &LogPvDataDaily{}
	if *date == "" {
		tmp := time.Now().Format(keyDateFormat)
		date = &tmp
	}
	l.LogKey = fmt.Sprintf("%s", *date)
	plantkey, err := g.GetStructKey(plant)
	if err != nil {
		c.Infof("Got an error when getting PvLogDaily from datastore, %s", err.Error())
	}
	l.Parent = plantkey
	err = g.Get(l)
	if err != nil {
		c.Infof("Got an error when getting PvLogDaily from datastore, %s", err.Error())
	}
	return l
}

func putLogDaily(plant *plantdata.Plant, date time.Time, r *http.Request, l *LogPvDataDaily) error {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)

	// l.LogKey = fmt.Sprintf("%s/%s", plant.PlantKey, time.Now().Format(keyDateFormat))
	// err := g.Put(l)
	// if err != nil {
	// 	c.Infof("Got an error when putting PvDataDaily to datastore, %s", err.Error())
	// 	return err
	// }

	// Put with plant as ancestor (parent)
	l.LogKey = fmt.Sprintf("%s", date.Format(keyDateFormat))
	plantkey, err := g.GetStructKey(plant)
	l.Parent = plantkey
	if err != nil {
		c.Infof("Got an error when putting PvDataDaily to datastore, %s", err.Error())
		return err
	}
	err = g.Put(l)
	if err != nil {
		c.Infof("Got an error when putting PvDataDaily to datastore, %s", err.Error())
		return err
	}
	return nil
}

// func moveOldPvLogDaily(w http.ResponseWriter, r *http.Request) {
// 	// All plants
// 	c := appengine.NewContext(r)
// 	g := goon.NewGoon(r)
// 	q := logDailyGetPlants(w, r)

// 	for i := g.Run(q); ; {
// 		p := plantdata.Plant{}
// 		_, err := i.Next(&p)
// 		if err == datastore.Done {
// 			break
// 		}
// 		if err != nil {
// 			c.Infof("Moveoldpvlog: Could not list plants for dailylog from datastore due to %s", err.Error())
// 			http.Error(w, fmt.Sprintf(err.Error()),
// 				http.StatusInternalServerError)
// 			return
// 		}

// 		// OK p is now a plant.
// 		// Go through dates, start at 20130715
// 		//curdate := getDate(2013, 07, 15)
// 		enddate := getDate(2013, time.August, 18)
// 		for curdate := getDate(2013, time.July, 15); curdate.Before(enddate); curdate = curdate.AddDate(0, 0, 1) {
// 			// Get the old log
// 			l := &LogPvDataDaily{}
// 			l.LogKey = fmt.Sprintf("%s/%s", p.PlantKey, curdate.Format(keyDateFormat))
// 			err = g.Get(l)
// 			if err != nil {
// 				c.Infof("Moveoldpvlog: Got an error when getting PvLogDaily from datastore, %s", err.Error())
// 				continue
// 			}
// 			c.Debugf("Now moving pvlog for plant %s, for date %s...",
// 				p.PlantKey, curdate.Format(keyDateFormat))

// 			newl := &LogPvDataDaily{}

// 			newl.Logs = l.Logs

// 			//Put the new log
// 			err = putLogDaily(&p, curdate, r, newl)
// 			if err != nil {
// 				c.Infof("Moveoldpvlog: Got an error when puttinh PvLogDaily to datastore, %s", err.Error())
// 			} else {
// 				c.Debugf("Finished moving pvlog for plant %s, for date %s", p.PlantKey,
// 					curdate.Format(keyDateFormat))
// 			}

// 			//curdate = curdate.AddDate(0, 0, 1)
// 		}

// 	}

// }

func getDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
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
