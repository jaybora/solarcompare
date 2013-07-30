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

// This is for the marshalling to json
type LogPvDataForJson struct {
	LogTime time.Time
	PvData  dataproviders.PvData
}

// This is stored in the datastore
type LogPvDataDaily struct {
	LogKey string `goon:"id"`
	Logs   []byte //Json encoded array of LogPvDataForJson
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

	if r.URL.Query().Get("logtype") == "daily" {
		logDaily(w, r)
	} else {
		keypos := 2
		if !appengine.IsDevAppServer() {
			keypos += 2
		}

		plantkey := web.PlantKey(r.URL.String(), keypos)
		if plantkey != "" {
			logdatadaily := getLogDaily(&plantkey, r)
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

		logDaily := getLogDaily(&p.PlantKey, r)
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
		newLog := LogPvDataForJson{time.Now(), pvdata}
		newLogs := append(exsistingLogs, newLog)

		// Make it a json string again
		b, err := json.MarshalIndent(&newLogs, "", "  ")
		if err != nil {
			c.Errorf("ERROR on mashalling logpvdata to JSON: %s", err.Error())
		}
		logDaily.Logs = b

		//Put it in store
		putLogDaily(&p.PlantKey, r, logDaily)
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

func getLogDaily(plantkey *string, r *http.Request) *LogPvDataDaily {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)

	l := &LogPvDataDaily{}
	l.LogKey = fmt.Sprintf("%s/%s", *plantkey, time.Now().Format(keyDateFormat))
	err := g.Get(l)
	if err != nil {
		c.Infof("Got an error when getting PvDataDaily from datastore, %s", err.Error())
	}
	return l
}

func putLogDaily(plantkey *string, r *http.Request, l *LogPvDataDaily) error {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)

	l.LogKey = fmt.Sprintf("%s/%s", *plantkey, time.Now().Format(keyDateFormat))
	err := g.Put(l)
	if err != nil {
		c.Infof("Got an error when putting PvDataDaily to datastore, %s", err.Error())
		return err
	}
	return nil
}
