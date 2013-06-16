// +build appengine
package handlers
 
import (
	"net/http"
	"web"
	"dataproviders"
	"fmt"
	"time"
	"bytes"
	"io/ioutil"
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)
  


func Pvdatahandler(w http.ResponseWriter, r *http.Request) {
	plantkey := web.PlantKey(r.URL.String(), appengine.IsDevAppServer())
	if plantkey == "" {
		http.Error(w, fmt.Sprintf("No plant specified"), 
			http.StatusBadRequest)
		return	
	}
	
    switch r.Method {
        case "GET": handleGetPvData(w, r, &plantkey)
        case "POST": handlePutPvData(w, r, &plantkey)
        case "PUT": handlePutPvData(w, r, &plantkey)
//        case "DELETE": handleDelete(w, r, &plantkey)
    }
}

func keyPvData(plantkey *string) string {
	return "pvdata:" + *plantkey
}

func handleGetPvData(w http.ResponseWriter, r *http.Request, 
        plantkey *string) {
	c := appengine.NewContext(r)
	
	i, err := memcache.Get(c, keyPvData(plantkey));
	if err != nil {
		c.Errorf("Could not get memcache for %s due to %s", *plantkey, err.Error())
		http.Error(w, fmt.Sprintf("No data for plant %s is available. (%s). Signaling request for data on that plant...", *plantkey, err.Error()), 
			http.StatusNotFound)
		startDpUnit(r, plantkey)
		return
	}
	if checkForRestart(c, i.Value) {
		startDpUnit(r, plantkey)
	}
	w.Write(i.Value)
}

func handlePutPvData(w http.ResponseWriter, r *http.Request, plantkey *string) {
	c := appengine.NewContext(r)
	jsonbytes, _ := ioutil.ReadAll(r.Body)
	
	item := &memcache.Item{
    Key:   keyPvData(plantkey),
    Value: jsonbytes}
	
	err := memcache.Set(c, item);
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
		
	c.Debugf("Ret is %s. Since %s is before %s", ret, pvdata.LatestUpdate, treshold)
	return ret;
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
		c.Errorf("Plant %s not found in startDpUnit. Should not happen. %s", *plantkey, err.Error())
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