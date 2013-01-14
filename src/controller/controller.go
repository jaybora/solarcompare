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
	"log"
	"plantdata"
	"sync"
)

// Locker for sync'ing the live map
var lock = sync.RWMutex{}

// The map where the live dataproviders are kept
type Controller struct {
	live map[string]dataproviders.DataProvider
}

// Create a new controller
// Only one for entire app
func NewController() Controller {
	return Controller{map[string]dataproviders.DataProvider{}}
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
		err = c.startNewProvider(plantdata)
		if err != nil {
			return
		}
		lock.RLock()
		provider, _ = c.live[plantdata.PlantKey]
		lock.RUnlock()
		return
	}
	return
}

func (c *Controller) startNewProvider(plantdata *plantdata.PlantData) error {
	json, _ := plantdata.ToJson()
	log.Printf("Starting new dataprovider for plant %s", json)

	p, err := dispatcher.Provider(plantdata.DataProvider,
		plantdata.InitiateData,
		func() {
			c.providerTerminated(plantdata.PlantKey)
		})
	if err != nil {
		return err
	}
	lock.Lock()
	c.live[plantdata.PlantKey] = p
	lock.Unlock()
	return nil

}

func (c *Controller) providerTerminated(plantKey string) {
	log.Printf("Controller, plantkey %s gone offline", plantKey)
	lock.Lock()
	delete(c.live, plantKey)
	lock.Unlock()
}
