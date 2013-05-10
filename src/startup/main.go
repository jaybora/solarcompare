// +build !appengine

package main

import (
	"controller"
	"net/http"
	"web"
	"plantdata"
	"dataproviders"
	"encoding/json"
	"logger"
	"time"
	"httpclient"
	"io/ioutil"
	"sync"
)

import _ "net/http/pprof"

var log = logger.NewLogger(logger.INFO, "main: ")

type staticPlants struct {
	plants map[string]plantdata.PlantData
}

// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY,
	                           InitiateData: dataproviders.InitiateData{PlantKey: "jbr"}},
	"peterlarsen": plantdata.PlantData{PlantKey: "peterlarsen", 
	                           Name: "Guldnældevænget 35",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"peterlarsen", "jesper@jbr.dk", "cidaxura", "3", ""}},
	"kaup": plantdata.PlantData{PlantKey: "kaup", 
	                           Name: "Pandebjergvej",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"kaup", "jesper@jbr.dk", "cidaxura", "2", ""}},
	"gldv33": plantdata.PlantData{PlantKey: "gldv33", 
	                           Name: "Guldnældevænget 33",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"gldv33", "jesper@jbr.dk", "cidaxura", "1", ""}},
	"janbang": plantdata.PlantData{PlantKey: "janbang", 
	                           Name: "Fuglehaven",
	                           DataProvider: dataproviders.Danfoss,
	                           InitiateData: dataproviders.InitiateData{"janbang", "anonym", "anonym", "", "5.103.131.3"}},
	"lysningen": plantdata.PlantData{PlantKey: "lysningen", 
	                           Name: "Janniks anlæg",
	                           DataProvider: dataproviders.Kostal,
	                           InitiateData: dataproviders.InitiateData{"lysningen", "pvserver", "2674", "", "2.104.143.225"}},
}


func main() {
	pvStore := NewPvStore()
	plants := staticPlants{plantmap}
	controller := controller.NewController(httpclient.NewClient, 
		 pvStore,
		 StatsStore{})
	http.HandleFunc("/", web.DefaultHandler)
	http.HandleFunc("/plant/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants, pvStore, true)
	})
	http.Handle("/scripts/",  http.FileServer(http.Dir(".")))
	http.Handle("/html/",  http.FileServer(http.Dir(".")))
	
	srv := http.Server{Addr: ":8090", ReadTimeout: 10*time.Second}

	err := srv.ListenAndServe()
	
	if err != nil {
		log.Info(err.Error())
	}
}

func (s staticPlants)PlantData(plantkey string) *plantdata.PlantData {
	log.Tracef("Getting plant for plantkey: %s", plantkey)
	plant, ok := s.plants[plantkey]
	if !ok {
		return nil
	}
	return &plant;
}

func (s staticPlants)ToJson() []byte {
	log.Tracef("Getting all plants as json")
	b, _ := json.MarshalIndent(&s.plants, "", "   ")
	return b;
}


type PvStore struct {
	// Locker for sync'ing the live map
	pvStoreLock sync.RWMutex
	pvDataMap map[string]dataproviders.PvData
}

func NewPvStore() PvStore {
	pvStoreLock := sync.RWMutex{}
	pvDataMap := map[string]dataproviders.PvData{}
	return PvStore{pvStoreLock, pvDataMap}
}

func (p PvStore)Set(plantkey string, pv *dataproviders.PvData) {
	p.pvStoreLock.Lock()
	p.pvDataMap[plantkey] = *pv
	p.pvStoreLock.Unlock()
}

func (p PvStore)Get(plantkey string) dataproviders.PvData {
	p.pvStoreLock.RLock()
	defer func() {
		p.pvStoreLock.RUnlock()
	}()
	return p.pvDataMap[plantkey]
} 


const Statfilename = "_stats.json"

type StatsStore struct {
}

// Load up stats from filesystem
func (s StatsStore)LoadStats(plantkey string) dataproviders.PlantStats {
	stats := dataproviders.PlantStats{}
	bytes, err := ioutil.ReadFile(plantkey + Statfilename)
	if err != nil {
		log.Infof("Error in reading statfile for plant %s: %s", plantkey, err.Error())
		return stats
	}
	err = json.Unmarshal(bytes, &stats)
	return stats
}

func (s StatsStore)SaveStats(plantkey string, pv *dataproviders.PvData) {
	stats := dataproviders.PlantStats{}
	stats.PowerAcPeakAll = pv.PowerAcPeakAll
	stats.PowerAcPeakAllTime = pv.PowerAcPeakAllTime
	stats.PowerAcPeakToday = pv.PowerAcPeakToday
	stats.PowerAcPeakTodayTime = pv.PowerAcPeakTodayTime
	bytes, err := json.Marshal(stats)
	if err != nil {
		log.Failf("Could not marshal plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
	err = ioutil.WriteFile(plantkey+Statfilename, bytes, 0777)
	if err != nil {
		log.Failf("Could not write plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
}

