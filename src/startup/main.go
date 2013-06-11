// +build !appengine

package main

import (
	"controller"
	"net/http"
	"web"
	"plantdata"
	"dataproviders"
	"logger"
	"time"
	"httpclient"
	"stores"
)

import _ "net/http/pprof"

var log = logger.NewLogger(logger.INFO, "main: ")


// Load up an example plant
var plantmap = map[string]plantdata.Plant {
	"jbr": plantdata.Plant{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY,
	                           InitiateData: dataproviders.InitiateData{PlantKey: "jbr"}},
	"peterlarsen": plantdata.Plant{PlantKey: "peterlarsen", 
	                           Name: "Guldnældevænget 35",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"peterlarsen", "jesper@jbr.dk", "cidaxura", "3", ""}},
	"kaup": plantdata.Plant{PlantKey: "kaup", 
	                           Name: "Pandebjergvej",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"kaup", "jesper@jbr.dk", "cidaxura", "2", ""}},
	"gldv33": plantdata.Plant{PlantKey: "gldv33", 
	                           Name: "Guldnældevænget 33",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"gldv33", "jesper@jbr.dk", "cidaxura", "1", ""}},
	"janbang": plantdata.Plant{PlantKey: "janbang", 
	                           Name: "Fuglehaven",
	                           DataProvider: dataproviders.Danfoss,
	                           InitiateData: dataproviders.InitiateData{"janbang", "anonym", "anonym", "", "5.103.131.3"}},
	"lysningen": plantdata.Plant{PlantKey: "lysningen", 
	                           Name: "Janniks anlæg",
	                           DataProvider: dataproviders.Kostal,
	                           InitiateData: dataproviders.InitiateData{"lysningen", "pvserver", "2674", "", "2.104.143.225"}},
}


func main() {
	pvStore := stores.NewPvStore()
	plants := stores.NewPlantStorePreloaded(plantmap)
	controller := controller.NewController(httpclient.NewClient, 
		 pvStore,
		 stores.StatsStoreFile{}, nil)
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


