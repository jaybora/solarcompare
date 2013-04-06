// +build appengine

package startup

import (
	"net/http"
	"logger"
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	"fmt"
	"web"
	"net/url"
	"dataproviders"
	"time"
)

var log = logger.NewLogger(logger.INFO, "app.go ")

func init() {
	http.HandleFunc("/plant/", planthandler)
}

func planthandler(w http.ResponseWriter, r *http.Request) {
	plantkey := web.PlantKey(r.URL.String(), appengine.IsDevAppServer())
	if plantkey == "" {
		http.Error(w, fmt.Sprintf("No plant specified"), 
			http.StatusNotFound)
		return	
	}
	c := appengine.NewContext(r)
	
	i, err := memcache.Get(c, "pvdata:"+plantkey);
	if err != nil {
		c.Errorf("Could not get memcache for %s due to %s", plantkey, err.Error())
		http.Error(w, fmt.Sprintf("No data for plant %s is available. (%s). Starting datagrabber....", plantkey, err.Error()), 
			http.StatusNotFound)
		startGrabber(c, plantkey)
		return
	}
	if checkForRestart(c, i.Value) {
		startGrabber(c, plantkey)
	}
	w.Write(i.Value)
}

func checkForRestart(c appengine.Context, jsonPvData []byte) bool {
	c.Debugf("Jsondata from memcache: %s", jsonPvData)
	pvdata := dataproviders.FromJson(jsonPvData)
	treshold := time.Now().UTC().Add(-1 * time.Minute)
	
	ret := pvdata.LatestUpdate == nil ||
		pvdata.LatestUpdate.Before(treshold)
		
	c.Debugf("Ret is %s. Since %s is before %s", ret, pvdata.LatestUpdate, treshold)
	return ret;
}

func backend(c appengine.Context) string {
	// Use the load-balanced hostname for the "datagrabber" backend.
	return "http://" + appengine.BackendHostname(c, "datagrabber", -1)
}

func startGrabber(c appengine.Context, plantkey string) {
	u := backend(c) + "/plantsbackend/" + url.QueryEscape(plantkey)
	c.Infof("Sending setup request to datagrabber for plant %s", plantkey)
	resp, err := urlfetch.Client(c).Get(u)
	if err != nil {
		c.Errorf("Could not start datagrabber for %s due to %s", plantkey, err.Error())		
	}
	defer resp.Body.Close()
}