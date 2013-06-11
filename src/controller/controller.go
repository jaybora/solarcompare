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
)

var log = logger.NewLogger(logger.TRACE, "Controller: ")

// Locker for sync'ing the live map
var lock = sync.RWMutex{}

// The map where the live dataproviders are kept
type Controller struct {
	live map[string]chan int
	newClient dispatcher.NewClient
	pvStore dataproviders.PvStore
	statsStore dataproviders.PlantStatsStore
	terminateCallback func(plantKey *string)
}

// Create a new controller
// Only one for entire app
func NewController(newClient dispatcher.NewClient, 
                   pvStore dataproviders.PvStore,
                   statsStore dataproviders.PlantStatsStore,
                   terminateCallback func(plantKey *string)) Controller {
	c := Controller{map[string]chan int{}, 
	                newClient, 
	                pvStore,
	                statsStore,
	                terminateCallback}
	//go printStatus(&c)
	return c
}

// Get the channel for the live provider of the given plantkey
// If the plant is not live, the controller will start a 
// new dataprovider
func (c *Controller) Provider(plantdata *plantdata.Plant) (err error) {
	lock.RLock()
	_, ok := c.live[plantdata.PlantKey]
	lock.RUnlock()
	if ok {
		return
	} else {
		// No live, startup a new provider
		// Lock again to prevent that multiple providers would be started
		lock.Lock();
		// Look again if someone else has started the provider
		_, ok = c.live[plantdata.PlantKey]
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
		//provider, _ = c.live[plantdata.PlantKey]
		lock.Unlock()
		return
	}
	return
}

func (c *Controller) Terminate(plantdata *plantdata.Plant) {
	lock.RLock()
	terminateCh, ok := c.live[plantdata.PlantKey]
	lock.RUnlock()
	if ok {
		// Signal terminate on channel to dataprovider
		log.Debugf("Request for termination of plant %s", plantdata.PlantKey)
		terminateCh <- 1;
	} else {
		log.Infof("Could not terminate plant %s, as it is not found", plantdata.PlantKey)
	}
}

//
//func printStatus(c *Controller) {
//	tick := time.NewTicker(1 * time.Minute)
//	tickCh := tick.C
//	
//	for {
//		<-tickCh
//		log.Info("List of online plants:")
//		log.Info("----------------------------------------------------")
//		lock.RLock();
//		for k, v := range c.live {
//			pvdata, _ := v.PvData()
//			if pvdata.LatestUpdate == nil {
//				log.Infof(" - %s, no update yet", k)
//			} else {
//				log.Infof(" - %s, latest update at %s", k, pvdata.LatestUpdate.Format(time.RFC822))
//			}
//		}
//		lock.RUnlock();
//		log.Info("----------------------------------------------------")
//	}
//
//
//}

func (c *Controller) startNewProvider(plantdata *plantdata.Plant) error {
	json, _ := plantdata.ToJson()
	log.Infof("Starting new dataprovider for plant %s", json)
	
	terminateCh := make(chan int)

	err := dispatcher.Provider(plantdata.DataProvider,
		plantdata.InitiateData,
		func() {
			c.providerTerminated(plantdata.PlantKey)
		}, 
		terminateCh,
		c.newClient,
		c.pvStore,
		c.statsStore)
	if err != nil {
		return err
	}
	//Adding terminate ch. Signaling this will terminate the dataprovider
	c.live[plantdata.PlantKey] = terminateCh
	return nil

}

func (c *Controller) providerTerminated(plantKey string) {
	log.Infof("Controller, plantkey %s gone offline", plantKey)
	lock.Lock()
	delete(c.live, plantKey)
	lock.Unlock()
	c.terminateCallback(&plantKey)
}
