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
)

var log = logger.NewLogger(logger.INFO, "main: ")

type staticPlants struct {
	plants map[string]plantdata.PlantData
}

// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY},
	"peterlarsen": plantdata.PlantData{PlantKey: "peterlarsen", 
	                           Name: "Guldnældevænget 35",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"jesper@jbr.dk", "cidaxura", "3"}},
	"kaup": plantdata.PlantData{PlantKey: "kaup", 
	                           Name: "Pandebjergvej",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"jesper@jbr.dk", "cidaxura", "2"}},
	"gldv33": plantdata.PlantData{PlantKey: "gldv33", 
	                           Name: "Guldnældevænget 33",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"jesper@jbr.dk", "cidaxura", "1"}},
	"lysningen": plantdata.PlantData{PlantKey: "lysningen", 
	                           Name: "Janniks anlæg",
	                           DataProvider: dataproviders.Suntrol,
	                           InitiateData: dataproviders.InitiateData{PlantNo: "7982"}},
}

func main() {
	plants := staticPlants{plantmap}
	controller := controller.NewController()

	http.HandleFunc("/", web.DefaultHandler)
	http.HandleFunc("/plants/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants)
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
