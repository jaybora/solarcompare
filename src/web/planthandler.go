package web

import (
	"fmt"
	"net/http"
	"strings"
	"controller"
	"plantdata"
)

type PlantDataGetter interface {
	PlantData(plantkey string) *plantdata.PlantData
	ToJson() []byte
}

func PlantHandler(w http.ResponseWriter, r *http.Request, 
		c *controller.Controller, pg PlantDataGetter) {
	plantkey := plantKey(r.URL.String())
	if plantkey == "" {
		listPlants(w, pg)
		return
	}
    
    
    //Lookup plantdata	
    plantdata := pg.PlantData(plantkey)
    
    if plantdata == nil {
    	err := fmt.Errorf("404: Plant %s not found", plantkey)
    	http.Error(w, err.Error(), http.StatusNotFound)
    	return
    }
    
    // Go to the controller with the plantdata
    // The controller will start up a plant service if its not allready live
    // We get the provider from the controller
    provider, err := c.Provider(plantdata)
    if err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }
    
	w.Header().Set("Content-Type", "application/json")
	pvdata, err := provider.PvData()
    if err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }
	
	w.Write(pvdata.ToJson())
    
}

// List all known plants if no plantkey is given
func listPlants(w http.ResponseWriter, pg PlantDataGetter) {
	w.Header().Set("Content-Type", "application/json;  charset=utf-8")
	w.Write(pg.ToJson())
} 

func plantKey(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 2 {
		return strings.Split(parts[2], "?")[0]
	}
	return ""
	
}