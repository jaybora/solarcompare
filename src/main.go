package main

import (
	"controller"
	"net/http"
	"web"
	"plantdata"
	"dataproviders"
	"log"
)

type staticPlants struct {
	plants map[string]plantdata.PlantData
}

// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY},
}

func main() {
	plants := staticPlants{plantmap}
	controller := controller.NewController()

	http.HandleFunc("/", web.DefaultHandler)
	http.HandleFunc("/plant/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants)
	})
	http.ListenAndServe(":8080", nil)
}

func (s staticPlants)PlantData(plantkey string) *plantdata.PlantData {
	log.Printf("Getting plant for plantkey: %s", plantkey)
	plant, ok := s.plants[plantkey]
	if !ok {
		return nil
	}
	return &plant;
}