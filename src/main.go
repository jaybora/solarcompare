package main

import (
	"controller"
	"net/http"
	"web"
	"plantdata"
	"dataproviders"
	"log"
	"encoding/json"
)

type staticPlants struct {
	plants map[string]plantdata.PlantData
}

// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY},
	"peterlarsen": plantdata.PlantData{PlantKey: "peterlarsen", 
	                           Name: "Guldnældevænget 5",
	                           DataProvider: dataproviders.SunnyPortal,
	                           InitiateData: dataproviders.InitiateData{"jesper@jbr.dk", "cidaxura", "2"}},
}

func main() {
	plants := staticPlants{plantmap}
	controller := controller.NewController()

	http.HandleFunc("/", web.DefaultHandler)
	http.HandleFunc("/plants/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants)
	})
	http.ListenAndServe(":8090", nil)
}

func (s staticPlants)PlantData(plantkey string) *plantdata.PlantData {
	log.Printf("Getting plant for plantkey: %s", plantkey)
	plant, ok := s.plants[plantkey]
	if !ok {
		return nil
	}
	return &plant;
}

func (s staticPlants)ToJson() []byte {
	log.Print("Getting all plants as json")
	b, _ := json.MarshalIndent(&s.plants, "", "   ")
	return b;
}
