package stores

import (
	"sync"
	"dataproviders"
)


type PvStore struct {
	// Locker for sync'ing the live map
	pvStoreLock sync.RWMutex
	pvDataMap map[string]dataproviders.PvData
}

func NewPvStore() PvStore {
	pvStoreLock := sync.RWMutex{}
	pvDataMap := map[string]dataproviders.PvData{}
	return PvStore{pvStoreLock, pvDataMap}
}

func (p PvStore)Set(plantkey string, pv *dataproviders.PvData) {
	p.pvStoreLock.Lock()
	p.pvDataMap[plantkey] = *pv
	p.pvStoreLock.Unlock()
}

func (p PvStore)Get(plantkey string) dataproviders.PvData {
	p.pvStoreLock.RLock()
	defer func() {
		p.pvStoreLock.RUnlock()
	}()
	return p.pvDataMap[plantkey]
} 
