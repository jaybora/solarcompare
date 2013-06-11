package stores

import (
	"encoding/json"
	"plantdata"
	"logger"
	"sync"
)

var log = logger.NewLogger(logger.INFO, "plantstore: ")


type PlantStore struct {
	// Locker for sync'ing the map
	storeLock sync.RWMutex
	plants map[string]plantdata.Plant
}

func NewPlantStore() PlantStore {
	storeLock := sync.RWMutex{}
	plants := map[string]plantdata.Plant{}
	return PlantStore{storeLock, plants}
}

// Constructor with preloaded map
func NewPlantStorePreloaded(plants map[string]plantdata.Plant) PlantStore {
	storeLock := sync.RWMutex{}
	return PlantStore{storeLock, plants}
}


func (p PlantStore)Remove(plantkey string) {
	p.storeLock.Lock()
	delete(p.plants, plantkey)
	p.storeLock.Unlock()
}

func (p PlantStore)Add(plantkey string, plant *plantdata.Plant) {
	p.storeLock.Lock()
	p.plants[plantkey] = *plant
	p.storeLock.Unlock()
}

func (s PlantStore)Get(plantkey string) *plantdata.Plant {
	log.Tracef("Getting plant for plantkey: %s", plantkey)
	s.storeLock.RLock()
	defer func() {
		s.storeLock.RUnlock()
	}()

	plant, ok := s.plants[plantkey]
	if !ok {
		return nil
	}
	return &plant;
}

func (s PlantStore)ToJson() []byte {
	log.Tracef("Getting all plants as json")
	s.storeLock.RLock()
	defer func() {
		s.storeLock.RUnlock()
	}()
	b, _ := json.MarshalIndent(&s.plants, "", "   ")
	return b;
}


