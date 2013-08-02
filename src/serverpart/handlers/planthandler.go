// +build appengine
package handlers

// This handler does GRUD over plants via REST json to app engine datastore

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"fmt"
	"goon"
	"io/ioutil"
	"net/http"
	"plantdata"
	"strings"
	"time"
	"web"
)

func PlantHandler(w http.ResponseWriter, r *http.Request) {
	// If pvdata is specified then go to that handler
	if strings.Contains(r.URL.String(), "/pvdata") {
		Pvdatahandler(w, r)
		return
	}
	if strings.Contains(r.URL.String(), "/logpvdata") {
		LogPvDataHandler(w, r)
		return
	}
	keypos := 2
	if !appengine.IsDevAppServer() {
		keypos += 2
	}

	plantkey := web.PlantKey(r.URL.String(), keypos)
	if plantkey == "" {
		handleListPlants(w, r)
		return
	}

	switch r.Method {
	case "GET":
		handleGetPlant(w, r, &plantkey)
	case "POST":
		handlePutPlant(w, r, &plantkey)
	case "PUT":
		handlePutPlant(w, r, &plantkey)
	case "DELETE":
		handleDeletePlant(w, r, &plantkey)
	}
}

func handleGetMultiplePlants(w http.ResponseWriter, r *http.Request, keys *[]string) {
	c := appengine.NewContext(r)

	// Manually buliding json array, outputting one element at a time
	any := false
	for index, plantkey := range *keys {
		any = true
		c.Debugf("Getting plant for key %s", plantkey)
		p, err := Plant(r, &plantkey)
		if index == 0 {
			w.Header().Add("Content-type", "application/json")
			w.Write([]byte("["))
		} else {
			w.Write([]byte(","))
		}
		if err != nil {
			c.Infof("Could not get plant for %s from datastore due to %s", plantkey, err.Error())
			w.Write([]byte("{}"))
		} else {
			json, err := p.ToJson()
			if err != nil {
				c.Infof("Error in marshalling to json, err %s", err.Error())
				w.Write([]byte("{}"))
			}
			w.Write(json)
		}
	}
	if any {
		w.Write([]byte("]"))
	}
}

/*
 List all plants from the logged in user
*/
func handleListPlants(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	u := user.Current(c)

	// Get multiple
	plantkeys := strings.Split(r.URL.Query().Get("plants"), ",")
	c.Debugf("Getting plants for given PlantKeys %s, antal %s", plantkeys, len(plantkeys))
	if r.URL.Query().Get("plants") != "" {
		handleGetMultiplePlants(w, r, &plantkeys)
		return
	}

	var q *datastore.Query
	// Request for the users plants only
	if r.URL.Query().Get("myplants") == "true" {
		if u == nil {
			http.Error(w, "Please login", http.StatusForbidden)
			return
		}
		c.Debugf("Getting list of plants for user %s", u.String())
		q = datastore.NewQuery("Plant").
			Filter("User =", u.ID).
			Order("PlantKey")
	} else {
		// Limit to 100 for now. Should be impl with url params for start and limit
		c.Debugf("Getting list of all plants")
		q = datastore.NewQuery("Plant").
			Order("PlantKey").
			Limit(100)
	}

	// Manually buliding json array, outputting one element at a time
	index := 0
	for i := g.Run(q); ; {
		p := plantdata.Plant{}
		_, err := i.Next(&p)
		if err == datastore.Done {
			break
		}
		if err != nil {
			if _, ok := err.(*datastore.ErrFieldMismatch); ok {
				//Ignore that error
				c.Debugf("Ignoring %s on load for myplants", err.Error())
			} else {
				c.Infof("Could not list myplants for user %s from datastore due to %s", u.String(), err.Error())
				http.Error(w, fmt.Sprintf(err.Error()),
					http.StatusInternalServerError)
				return
			}
		}
		if index == 0 {
			w.Header().Add("Content-type", "application/json")
			w.Write([]byte("["))
		} else {
			w.Write([]byte(","))
		}

		json, err := p.ToJson()
		if err != nil {
			c.Infof("Error in marshalling to json, err %s", err.Error())
		}
		w.Write(json)

		index++

	}
	if index > 0 {
		w.Write([]byte("]"))
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
		if err == datastore.ErrNoSuchEntity {
			c.Infof("Could not get plant for %s from datastore due to %s", *plantkey, err.Error())
			http.Error(w, fmt.Sprintf("Plant not found"),
				http.StatusNotFound)
			return
		} else {
			c.Infof("Errer %s when getting plant %s from datastore. That was ignored",
				err.Error(), *plantkey)
		}

	}
	json, err := ps.ToJson()
	if err != nil {
		c.Infof("Error in marshalling to json for plantkey %s, err %s", *plantkey, err.Error())
	}
	w.Header().Add("Content-type", "application/json")
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
	if u == nil {
		http.Error(w, "Please login", http.StatusForbidden)
		return
	}
	ps.User = u.ID

	// Check if allready exists, and only allow overwrite if user.ID is equal
	existingPlant, err := Plant(r, plantkey)
	if err == nil && existingPlant.User != u.ID {
		c.Errorf("Could not write plant data for plant as plantkey allready taking by another user %s: %s", *plantkey, ps)
		http.Error(w, fmt.Sprintf("Plant data could be stored. PlantKey %s is owned by some else", *plantkey),
			http.StatusForbidden)
		return
	}

	if err := g.Put(&ps); err != nil {
		c.Errorf("Could not write plant data for plant %s: %s, %s", *plantkey, ps, err.Error())
		http.Error(w, fmt.Sprintf("Plant data could be stored"),
			http.StatusInternalServerError)
		return
	}
	handleGetPlant(w, r, plantkey)

}
func handleDeletePlant(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	u := user.Current(c)
	if u == nil {
		http.Error(w, "Please login", http.StatusForbidden)
		return
	}

	// Check if allready exists, and only allow delete if user.ID is equal
	existingPlant, err := Plant(r, plantkey)
	if err == nil && existingPlant.User != u.ID {
		c.Errorf("Could not delete plant data for plant as plantkey allready taking by another user %s: %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("Plant data could be deleted. PlantKey %s is owned by some else", *plantkey),
			http.StatusForbidden)
		return
	}

	if err := g.Delete(g.Key(&plantdata.Plant{PlantKey: *plantkey})); err != nil {
		c.Errorf("Could not delete plant %s: %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("Plant could not be deleted"),
			http.StatusInternalServerError)
		return
	}
	// Sometimes the plant is not deleted right away
	time.Sleep(1 * time.Second)
	w.Write([]byte("Ok"))
}
