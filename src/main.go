package main

import (
	"controller"
	"net/http"
	"web"
	"plantdata"
	"dataproviders"
)


// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY},
}

func main() {
	controller := controller.NewController()

	http.HandleFunc("/", web.DefaultHandler)
	http.HandleFunc("/plant/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, )
	})
	http.ListenAndServe(":8080", nil)
}

func PlantData(plantkey string) plantdata.PlantData {
	return plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY}
}