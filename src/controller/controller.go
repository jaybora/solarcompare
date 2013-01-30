package controller

// The controller will control which plants there is live
// With live it means that the dataprovider for the plant is actively pulling data from the plant
// The controller will launch new dataprovider one per plant
// The dataprovider will cache its last data, and return those data on request
// The controller is loaded once, and then accessed by all http requests,
// so we need to be threadsafe. The live map must be syncronized.

import (
	"dataproviders"
	"dataproviders/dispatcher"
	"plantdata"
	"sync"
	"logger"
	"time"
)

var log = logger.NewLogger(logger.TRACE, "Controller: ")

// Locker for sync'ing the live map
var lock = sync.RWMutex{}

// The map where the live dataproviders are kept
type Controller struct {
	live map[string]dataproviders.DataProvider
}

// Create a new controller
// Only one for entire app
func NewController() Controller {
	c := Controller{map[string]dataproviders.DataProvider{}}
	go printStatus(&c)
	return c
}

// Get the channel for the live provider of the given plantkey
// If the plant is not live, the controller will start a 
// new dataprovider
func (c *Controller) Provider(plantdata *plantdata.PlantData) (provider dataproviders.DataProvider, err error) {
	lock.RLock()
	provider, ok := c.live[plantdata.PlantKey]
	lock.RUnlock()
	if ok {
		return
	} else {
		// No live, startup a new provider
		// Lock again to prevent that multiple providers would be started
		lock.Lock();
		// Look again if someone else has started the provider
		provider, ok = c.live[plantdata.PlantKey]
		if ok {
			log.Infof("Someone else started the provider for plant %s", plantdata.PlantKey)
			lock.Unlock()
			return
		}
		err = c.startNewProvider(plantdata)
		if err != nil {
			log.Infof("Could not start provider for plant %s", plantdata.PlantKey)
			lock.Unlock()
			return
		}
		//lock.RLock()
		provider, _ = c.live[plantdata.PlantKey]
		lock.Unlock()
		return
	}
	return
}

func printStatus(c *Controller) {
	tick := time.NewTicker(1 * time.Minute)
	tickCh := tick.C
	
	for {
		<-tickCh
		log.Info("List of online plants:")
		log.Info("----------------------------------------------------")
		lock.RLock();
		for k, v := range c.live {
			pvdata, _ := v.PvData()
			log.Infof(" - %s, latest update at %s", k, pvdata.LatestUpdate.Format(time.RFC822))
		}
		lock.RUnlock();
		log.Info("----------------------------------------------------")
	}


}

func (c *Controller) startNewProvider(plantdata *plantdata.PlantData) error {
	json, _ := plantdata.ToJson()
	log.Infof("Starting new dataprovider for plant %s", json)

	p, err := dispatcher.Provider(plantdata.DataProvider,
		plantdata.InitiateData,
		func() {
			c.providerTerminated(plantdata.PlantKey)
		})
	if err != nil {
		return err
	}
	c.live[plantdata.PlantKey] = p
	return nil

}

func (c *Controller) providerTerminated(plantKey string) {
	log.Infof("Controller, plantkey %s gone offline", plantKey)
	lock.Lock()
	delete(c.live, plantKey)
	lock.Unlock()
}
