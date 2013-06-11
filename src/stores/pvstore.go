package stores

import (
	"sync"
	"dataproviders"
)


type PvStore struct {
	// Locker for sync'ing the live map
	pvStoreLock sync.RWMutex
	pvDataMap map[string]dataproviders.PvData
	setCallBack func(plantkey *string, pv *dataproviders.PvData)
}

func NewPvStore(setCallBack func(plantkey *string, pv *dataproviders.PvData)) PvStore {
	pvStoreLock := sync.RWMutex{}
	pvDataMap := map[string]dataproviders.PvData{}
	return PvStore{pvStoreLock, pvDataMap, setCallBack}
}

func (p PvStore)Set(plantkey string, pv *dataproviders.PvData) {
	p.pvStoreLock.Lock()
	p.pvDataMap[plantkey] = *pv
	p.pvStoreLock.Unlock()
	p.setCallBack(&plantkey, pv)
}

func (p PvStore)Get(plantkey string) dataproviders.PvData {
	p.pvStoreLock.RLock()
	defer func() {
		p.pvStoreLock.RUnlock()
	}()
	return p.pvDataMap[plantkey]
} 
