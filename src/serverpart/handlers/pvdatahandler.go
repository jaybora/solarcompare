// +build appengine
package handlers

import (
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	"bytes"
	"dataproviders"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"web"
)

func Pvdatahandler(w http.ResponseWriter, r *http.Request) {
	keypos := 2
	if !appengine.IsDevAppServer() {
		keypos += 2
	}

	plantkey := web.PlantKey(r.URL.String(), keypos)
	if plantkey == "" {
		plantkeys := strings.Split(r.URL.Query().Get("plants"), ",")
		if len(plantkeys) > 1 {
			handleGetMultiplePvData(w, r, &plantkeys)
		} else {
			http.Error(w, fmt.Sprintf("No plant specified"),
				http.StatusBadRequest)
		}
		return
	}

	switch r.Method {
	case "GET":
		handleGetPvData(w, r, &plantkey)
	case "POST":
		handlePutPvData(w, r, &plantkey)
	case "PUT":
		handlePutPvData(w, r, &plantkey)
		//        case "DELETE": handleDelete(w, r, &plantkey)
	}
}

func keyPvData(plantkey *string) string {
	return "pvdata:" + *plantkey
}

func handleGetMultiplePvData(w http.ResponseWriter, r *http.Request, keys *[]string) {
	c := appengine.NewContext(r)

	// Manually buliding json array, outputting one element at a time
	index := 0
	for index, plantkey := range *keys {
		i, err := memcache.Get(c, keyPvData(&plantkey))

		if index == 0 {
			w.Header().Add("Content-type", "application/json")
			w.Write([]byte("["))
		} else {
			w.Write([]byte(","))
		}

		if err != nil {
			c.Debugf("Could not get memcache for %s due to %s", plantkey, err.Error())

			// Do nothing than hint startup of dpunit
			startDpUnit(r, &plantkey)
			//Indicate empty object in json to keep realiable index i reply
			w.Write([]byte("{}"))
		} else {
			// The memcache has it allready as json
			w.Write(i.Value)
		}

		index++

	}
	if index > 0 {
		w.Write([]byte("]"))
	}
}

func handleGetPvData(w http.ResponseWriter, r *http.Request,
	plantkey *string) {
	c := appengine.NewContext(r)

	i, err := memcache.Get(c, keyPvData(plantkey))
	if err != nil {
		c.Infof("Could not get memcache for %s due to %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("No data for plant %s is available. (%s). Signaling request for data on that plant...", *plantkey, err.Error()),
			http.StatusNotFound)
		startDpUnit(r, plantkey)
		return
	}
	if checkForRestart(c, i.Value) {
		startDpUnit(r, plantkey)
	}
	w.Header().Add("Content-type", "application/json")
	w.Write(i.Value)
}

func handlePutPvData(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	jsonbytes, _ := ioutil.ReadAll(r.Body)

	item := &memcache.Item{
		Key:   keyPvData(plantkey),
		Value: jsonbytes}

	err := memcache.Set(c, item)
	if err != nil {
		c.Errorf("Could not set memcache for %s due to %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("ERROR: %s", err.Error()),
			http.StatusNotFound)
		return
	}
	w.Write([]byte("Ok"))
}

// Check for recent data in memcache
func checkForRestart(c appengine.Context, jsonPvData []byte) bool {
	c.Debugf("Jsondata from memcache: %s", jsonPvData)
	pvdata, _ := dataproviders.ToPvData(jsonPvData)
	treshold := time.Now().UTC().Add(-1 * time.Minute)

	ret := pvdata.LatestUpdate == nil ||
		pvdata.LatestUpdate.Before(treshold)

	c.Debugf("CheckForRestart of plant, returning %t. Since %s is before %s", ret, pvdata.LatestUpdate, treshold)
	return ret
}

func startDpUnit(r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	// Get a dpunit url
	dpu := DpUnitAvailable(r)
	if dpu == nil {
		c.Errorf("No available dataproviderunits! Cannot start plant %s", *plantkey)
		return
	}
	url := fmt.Sprintf("%s/%s", dpu.ConnectURL, *plantkey)

	// Get data for plant
	plant, err := Plant(r, plantkey)
	if err != nil {
		c.Errorf("Plant %s not found in startDpUnit. %s", *plantkey, err.Error())
		return
	}

	// Convert to json
	b, err := plant.ToJson()
	if err != nil {
		c.Errorf("Plant %s cannot be mashalled to json in startDpUnit. Should not happen", *plantkey)
		return
	}

	// Send order to dpunit
	c.Infof("Sending setup request to dpunit %s for plant %s", url, *plantkey)
	c.Debugf("Posting %s to %s", b, url)
	resp, err := urlfetch.Client(c).Post(url, "", bytes.NewBuffer(b))
	if err != nil {
		c.Errorf("Could not start dpunit for %s due to %s", *plantkey, err.Error())
		return
	}
	c.Debugf("Received a %s on that", resp.Status)
	if resp.StatusCode == 200 {
		// Make the plant go live on dpu
		resp.Body.Close()
		c.Debugf("Getting %s to make plant go online...", url)
		resp, err = urlfetch.Client(c).Get(url)
		c.Debugf("Received a %s on that", resp.Status)
	}
	defer resp.Body.Close()
}
