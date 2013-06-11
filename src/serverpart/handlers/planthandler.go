// +build appengine
package handlers
// This handler does GRUD over plants via REST json to app engine datastore

import (
	"net/http"
	"strings"
	"plantdata"
	"io/ioutil"
	"fmt"
	"web"
	"goon"
	"appengine"
	"appengine/user"
)

func PlantHandler(w http.ResponseWriter, r *http.Request) {
	plantkey := web.PlantKey(r.URL.String(), appengine.IsDevAppServer())
	if plantkey == "" {
		http.Error(w, fmt.Sprintf("No plant specified"), 
			http.StatusBadRequest)
		return	
	}
	
	// If pvdata is specified then go to that handler
	if strings.Contains(r.URL.String(), "pvdata") {
		Pvdatahandler(w, r)
		return
	}
	
    switch r.Method {
        case "GET": handleGetPlant(w, r, &plantkey)
        case "POST": handlePutPlant(w, r, &plantkey)
        case "PUT": handlePutPlant(w, r, &plantkey)
        case "DELETE": handleDeletePlant(w, r, &plantkey)
    }
}

/*
 Get a plant
 */
func Plant(r *http.Request, plantkey *string) (plant *plantdata.Plant, err error) {
	g := goon.NewGoon(r)
	plant = &plantdata.Plant{}
	plant.PlantKey = *plantkey
	err = g.Get(plant)
	return
}

func handleGetPlant(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	ps, err := Plant(r, plantkey)
		
	if err != nil {
		c.Infof("Could not get plant for %s from datastore due to %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("Plant not found"), 
			http.StatusNotFound)
		return
	}
	json, err := ps.ToJson()
	if err != nil {
		c.Infof("Error in marshalling to json for plantkey %s, err %s", *plantkey, err.Error())
	}
	w.Write(json)
}

func handlePutPlant(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	jsonbytes, _ := ioutil.ReadAll(r.Body)
	ps, err := plantdata.ToPlant(&jsonbytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Plant data could not be unmarshalled"), 
			http.StatusBadRequest)
		return		
	}
	
	ps.PlantKey = *plantkey
	u := user.Current(c)
	ps.User = u.ID

	if err := g.Put(&ps); err != nil {
		c.Errorf("Could not write plant data for plant %s: %s, %s", *plantkey, ps, err.Error())
		http.Error(w, fmt.Sprintf("Plant data could be stored"), 
			http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Ok"))
}
func handleDeletePlant(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	
	if err := g.Delete(g.Key(&plantdata.Plant{PlantKey:*plantkey})); err != nil {
		c.Errorf("Could not write delete plant %s: %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("Plant could not be deleted"), 
			http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Ok"))
}
