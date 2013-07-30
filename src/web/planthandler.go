package web

import (
	"bytes"
	"controller"
	"dataproviders"
	"encoding/json"
	"fmt"
	"logger"
	"net/http"
	"plantdata"
	"strings"
)

var log = logger.NewLogger(logger.DEBUG, "planthandler: ")

type PlantStore interface {
	Get(plantkey string) *plantdata.Plant
	Add(plantkey string, plant *plantdata.Plant)
	Remove(plantkey string)
	ToJson() []byte
}

func PlantHandler(w http.ResponseWriter, r *http.Request,
	c *controller.Controller,
	pg PlantStore,
	pvStore dataproviders.PvStore,
	plantkeyPosInUrl int) {
	plantkey := PlantKey(r.URL.String(), plantkeyPosInUrl)
	log.Debugf("Got request method %s on planthandler. URL was %s",
		r.Method, r.URL.String())
	if plantkey == "" {
		listPlants(w, pg)
		return
	}

	if c == nil {
		err := fmt.Errorf("Controller not started")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Is this a put request
	switch r.Method {
	case "GET":
		handleGet(w, r, &plantkey, c, pg, pvStore)
	case "POST":
		handlePut(w, r, &plantkey, c, pg, pvStore)
	case "PUT":
		handlePut(w, r, &plantkey, c, pg, pvStore)
	case "DELETE":
		handleDelete(w, r, &plantkey, c, pg, pvStore)
	}
}

// Add new plant, any existing will be overwritten
func handlePut(w http.ResponseWriter, r *http.Request,
	plantkey *string,
	c *controller.Controller,
	pg PlantStore,
	pvStore dataproviders.PvStore) {

	buffer := bytes.NewBuffer([]byte{})
	buffer.ReadFrom(r.Body)
	jsonBody := buffer.Bytes()
	plantdata := plantdata.Plant{}
	err := json.Unmarshal(jsonBody, &plantdata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pg.Add(*plantkey, &plantdata)

	w.Write([]byte("Ok. Plant added."))
}

func handleDelete(w http.ResponseWriter, r *http.Request,
	plantkey *string,
	c *controller.Controller,
	pg PlantStore,
	pvStore dataproviders.PvStore) {
	//Lookup plantdata
	plantdata := pg.Get(*plantkey)

	if plantdata == nil {
		err := fmt.Errorf("404: Plant %s not found", *plantkey)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Stop the dataprovider for the plant
	c.Terminate(plantdata)
	pg.Remove(*plantkey)

	w.Write([]byte("Ok. Plant removed"))
}

func handleGet(w http.ResponseWriter, r *http.Request,
	plantkey *string,
	c *controller.Controller,
	pg PlantStore,
	pvStore dataproviders.PvStore) {
	//Lookup plantdata
	log.Debugf("Getting data for plant %s from plantstore", *plantkey)
	plantdata := pg.Get(*plantkey)
	log.Debugf("Got data for plant %s from plantstore", plantdata)

	if plantdata == nil {
		err := fmt.Errorf("404: Plant %s not found", *plantkey)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Go to the controller with the plantdata
	// The controller will start up a plant service if its not allready live
	// We get the provider from the controller
	log.Debugf("Calling provider for plant %s ", *plantkey)
	err := c.Provider(plantdata)
	if err != nil {
		log.Failf("Got error from controller %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	pvdata := pvStore.Get(*plantkey)
	//pvdata, err := provider.PvData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(pvdata.ToJson())

}

// List all known plants if no plantkey is given
func listPlants(w http.ResponseWriter, pg PlantStore) {
	log.Debug("Got request for list plants")
	w.Header().Set("Content-Type", "application/json;  charset=utf-8")
	w.Write(pg.ToJson())
}

func PlantKey(url string, keyposoffset int) string {
	keypos := 2 + keyposoffset
	parts := strings.Split(url, "/")
	if len(parts) > keypos {
		return strings.Split(parts[keypos], "?")[0]
	}
	return ""
}
