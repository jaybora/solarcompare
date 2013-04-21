// +build appengine

package startup

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/runtime"
	"appengine/urlfetch"
	"controller"
	"dataproviders"
	"encoding/json"
	"io"
	"logger"
	"net/http"
	"net/url"
	"plantdata"
	"time"
	"web"
	"sync"
)

var log = logger.NewLogger(logger.INFO, "main: ")

type staticPlants struct {
	plants map[string]plantdata.PlantData
}

// Load up an example plant
var plantmap = map[string]plantdata.PlantData {
	"jbr": plantdata.PlantData{PlantKey: "jbr", 
	                           Name: "Klarinetvej 25",
	                           DataProvider: dataproviders.FJY,
	                           InitiateData: dataproviders.InitiateData{PlantKey: "jbr"}},
//	"peterlarsen": plantdata.PlantData{PlantKey: "peterlarsen", 
//	                           Name: "Guldnældevænget 35",
//	                           DataProvider: dataproviders.SunnyPortal,
//	                           InitiateData: dataproviders.InitiateData{"peterlarsen", "jesper@jbr.dk", "cidaxura", "3", ""}},
//	"kaup": plantdata.PlantData{PlantKey: "kaup", 
//	                           Name: "Pandebjergvej",
//	                           DataProvider: dataproviders.SunnyPortal,
//	                           InitiateData: dataproviders.InitiateData{"kaup", "jesper@jbr.dk", "cidaxura", "2",""}},
//	"gldv33": plantdata.PlantData{PlantKey: "gldv33", 
//	                           Name: "Guldnældevænget 33",
//	                           DataProvider: dataproviders.SunnyPortal,
//	                           InitiateData: dataproviders.InitiateData{"gldv33", "jesper@jbr.dk", "cidaxura", "1", ""}},
	"janbang": plantdata.PlantData{PlantKey: "janbang", 
	                           Name: "Fuglehaven",
	                           DataProvider: dataproviders.Danfoss,
	                           InitiateData: dataproviders.InitiateData{"janbang", "anonym", "anonym", "", "5.103.131.3"}},
	"lysningen": plantdata.PlantData{PlantKey: "lysningen", 
	                           Name: "Janniks anlæg",
	                           DataProvider: dataproviders.Suntrol,
	                           InitiateData: dataproviders.InitiateData{PlantKey: "lysningen", PlantNo: "7982"}},
}
var plants = staticPlants{plantmap}

// Locker for ctrlr as only one should be online at a time
var ctrlrlock = sync.RWMutex{}

var ctrlr *controller.Controller

var ctrlrfunc = func(c appengine.Context) {
	c.Infof("Checking for controller...")
	ctrlrlock.RLock()
	if ctrlr != nil {
		ctrlrlock.RUnlock()
		// Controller allready loaded. Do nothing then
		c.Debugf("Controller allready running. Did nothing then")
		return
	} else {
		c.Debugf("No controller was running, checking if someone else started a new controller...")
		// No controller, startup a new controller
		// Lock again to prevent that multiple controllers would be started
		ctrlrlock.RUnlock()
		ctrlrlock.Lock()
		// Look again if someone else has started the controller
		if ctrlr != nil {
			// Controller allready loaded. Do nothing then
			c.Debugf("Someone else started the controller. Did nothing then")
			ctrlrlock.Unlock()
			return
		} else {
			c.Infof("No controller was running, now starting a new one...")
			newctrlr := controller.NewController(
				func() *http.Client {
					jar := new(Jar)
					return &http.Client{
						Transport: &urlfetch.Transport{
							Context: c,
							AllowInvalidServerCertificate: true,
						},
						Jar: jar,
					}
				},
				func(plantkey string, pv dataproviders.PvData) {
					//Save pvdata to memcache
					//The latestupdate stamp is only updated
					//when running through channels in dataprovider
					//When running with memcache we have to this manually
					t := time.Now()
					pv.LatestUpdate = &t
					
					i := &memcache.Item{Key: "pvdata:" + plantkey, Value: pv.ToJson()}
					if err := memcache.Set(c, i); err != nil {
						c.Errorf("Could not set memcache for %s, to %s due to", plantkey, pv.ToJson(), err.Error())
					}
				},
				StatsStore{c})
			ctrlr = &newctrlr
			// Unlock so others will se the controller
			ctrlrlock.Unlock()
			c.Infof("Controller started")
			time.Sleep(29 * time.Minute)
			c.Infof("Controller stopping")
			ctrlr = nil
		}

	}
}

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		jar.cookies = append(jar.cookies, cookie)
	}
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

func init() {
	http.HandleFunc("/_ah/start", handleStart)
	http.HandleFunc("/_ah/stop", handleStop)
	http.HandleFunc("/plantsbackend/", func(w http.ResponseWriter, r *http.Request) {
		if ctrlr == nil {
			// We got a request while there where no ctrlr running. Restart then
			c := appengine.NewContext(r)
			runtime.RunInBackground(c, ctrlrfunc)
		}
		web.PlantHandler(w, r, ctrlr, plants, appengine.IsDevAppServer())
	})
}

func handleStart(w http.ResponseWriter, r *http.Request) {

	// This handler is executed when a backend instance is started.
	// If it responds with a HTTP 2xx or 404 response then it is ready to go.
	// Otherwise, the instance is terminated and restarted.
	// The instance will receive traffic after this handler returns.

	c := appengine.NewContext(r)
	c.Infof("Starting backend...")

	c.Infof("Backend started. /plantsbackend is now awailable")

	runtime.RunInBackground(c, ctrlrfunc)

	io.WriteString(w, "Ok, datagrabber backend started.")
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	// This handler is executed when a backend instance is being shut down.
	// It has 30s before it will be terminated.
	// When this is called, no new requests will reach the instance.

	c := appengine.NewContext(r)
	
	c.Infof("Backend stop request. Writing peaks to datastore...")
	for k, _ := range plantmap {
		i, err := memcache.Get(c, "pvdata:"+k)
		if err != nil {
			c.Debugf("Could not get pvdata from memcache for %s due to %s", k, err.Error())
			continue		
		}
		pv := dataproviders.PvData{}
		err = json.Unmarshal(i.Value, &pv)
		if err != nil {
			c.Infof("Could not unmarshal pvdata from memcache for %s due to %s", k, err.Error())
			continue
		}
		ss := StatsStore{c}
		ss.SaveStats(k, &pv)
	}
	
	
	c.Infof("Backend stopped.")
}

func (s staticPlants) PlantData(plantkey string) *plantdata.PlantData {
	log.Tracef("Getting plant for plantkey: %s", plantkey)
	plant, ok := s.plants[plantkey]
	if !ok {
		return nil
	}
	return &plant
}

func (s staticPlants) ToJson() []byte {
	log.Tracef("Getting all plants as json")
	b, _ := json.MarshalIndent(&s.plants, "", "   ")
	return b
}

type StatsStore struct {
	c appengine.Context
}

func (s StatsStore) Key(plantkey string) *datastore.Key {
	return datastore.NewKey(s.c, "plantstats", plantkey, 0, nil)
}

type DatastoreStats struct {
	Json string
}

// Load up stats from datastore
func (s StatsStore) LoadStats(plantkey string) dataproviders.PlantStats {
	ds := DatastoreStats{}
	if err := datastore.Get(s.c, s.Key(plantkey), &ds); err != nil {
		s.c.Infof("Could not get plantstats for %s from datastore due to %s", plantkey, err.Error())
	}
	
	stats := dataproviders.PlantStats{}
	if err := json.Unmarshal([]byte(ds.Json), &stats); err != nil {
		s.c.Infof("Could not umarshal plantstats for %s from datastore due to %s", plantkey, err.Error())
	}
	//s.c.Debugf("Loaded stats for %s, as %s", plantkey, ds.Json)
	return stats
}

func (s StatsStore) SaveStats(plantkey string, pv *dataproviders.PvData) {
	s.c.Infof("Saving stats for %s to datastore", plantkey)
	stats := dataproviders.PlantStats{}
	stats.PowerAcPeakAll = pv.PowerAcPeakAll
	stats.PowerAcPeakAllTime = pv.PowerAcPeakAllTime
	stats.PowerAcPeakToday = pv.PowerAcPeakToday
	stats.PowerAcPeakTodayTime = pv.PowerAcPeakTodayTime

	bytes, err := json.Marshal(stats)
	if err != nil {
		s.c.Errorf("Could not marshal plant stats for plant %s: %s", plantkey, err.Error())
		return
	}

	if _, err := datastore.Put(s.c, s.Key(plantkey), &DatastoreStats{string(bytes)}); err != nil {
		s.c.Errorf("Could not write plant stats for plant %s: %s, %s", plantkey, bytes, err.Error())
		return
	}
	//s.c.Debugf("Stats saved for %s as %s", plantkey, bytes)
}
