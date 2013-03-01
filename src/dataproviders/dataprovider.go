package dataproviders

import (
	"logger"
	"time"
	"io/ioutil"
	"encoding/json"

)

type PlantStats struct {
	PowerAcPeakAll       uint16
	PowerAcPeakAllTime   time.Time
	PowerAcPeakToday     uint16
	PowerAcPeakTodayTime time.Time
}

type InitiateData struct {
	UserName string
	Password string
	PlantNo  string
}

var log = logger.NewLogger(logger.DEBUG, "Dataprovider: generic: ")

type DataProvider interface {
	Name() string
	PvData() (pv PvData, err error)
}

type TerminateCallback func()

type UpdatePvData func(i *InitiateData, pv PvData) (pv PvData, err error)

// List of all known dataproviders
// This would be nice if we could autodetect them somehow
const (
	FJY         = iota
	SunnyPortal = iota
	Suntrol     = iota
)

const Statfilename = "_stats.json"

// Load up stats from filesystem
func loadStats(plantkey string) PlantStats {
	stats := PlantStats{}
	bytes, err := ioutil.ReadFile(plantkey + Statfilename)
	if err != nil {
		log.Infof("Error in reading statfile for plant %s: %s", plantkey, err.Error())
		return stats
	}
	err = json.Unmarshal(bytes, &stats)
	return stats;
}

func saveStats(plantkey string, pv *PvData) {
	stats := PlantStats{}
	stats.PowerAcPeakAll = pv.PowerAcPeakAll
	stats.PowerAcPeakAllTime = pv.PowerAcPeakAllTime
	stats.PowerAcPeakToday = pv.PowerAcPeakToday
	stats.PowerAcPeakTodayTime = pv.PowerAcPeakTodayTime
	bytes, err := json.Marshal(stats)
	if err != nil {
		log.Failf("Could not marshal plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
	err = ioutil.WriteFile(plantkey + Statfilename, bytes, 0777)
	if err != nil {
		log.Failf("Could not write plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
} 

// RunUpdates on the provider. 
// updateFast, a function that gets called when a fast update is scheduled
// updateSlow, a function that gets called when a slow update is scheduled
// fastTime, secs on updateFast should be scheduled
// slowTime, secs on updateSlow should be scheduled
// terminateTime, secs on how long the provider will stay online before it terminates
// updateCh, a channel to send updated PvData to
// reqCh, a channel to request actual PvData from
// term, a function that gets called when RunUpdates terminates
// termCh, a channel to signal when RunUpdates should terminate
// errClose, maximum number of errors received before giving up, and terminates
func RunUpdates(initiateData *InitiateData,
    updateFast UpdatePvData,
	updateSlow UpdatePvData,
	fastTime time.Duration,
	slowTime time.Duration,
	terminateTime time.Duration,
	updateCh chan PvData,
	reqCh chan chan PvData,
	term TerminateCallback,
	termCh chan int,
	errClose int,
	plantkey string) {
	log.Trace("Started a RunUpdates rutine")
	stats := loadStats(plantkey)
	
	

	// Fast Ticker
	fastTick := time.NewTicker(fastTime)
	fastTickCh := fastTick.C
	// Slow Ticker
	slowTick := time.NewTicker(slowTime)
	slowTickCh := slowTick.C
	// Terminate ticker
	terminateTicker := time.NewTicker(terminateTime)
	terminateCh := terminateTicker.C

	errCounter := 0
	firstRun := true

	shutdown := func() {
		log.Infof("About to terminate RunUpdates for plant %s", plantkey)
		fastTick.Stop()
		slowTick.Stop()
		terminateTicker.Stop()
		pvCh := make(chan PvData)
		reqCh <- pvCh
		pv := <-pvCh
		saveStats(plantkey, &pv) 
		term()
		termCh <- 0
		log.Infof("RunUpdates exited for plant %s", plantkey)
		return
	}

	defer func() {
		shutdown()
		if r := recover(); r != nil {
			log.Infof("Recovered in dataprovider.RunUpdates, %s", r)
		}
	}()

LOOP:
	for {
		// If this is first run, then reqCh will block
		// So we start with a zero'ed pv
		pv := PvData{}
		pv.PowerAcPeakAll = stats.PowerAcPeakAll
		pv.PowerAcPeakAllTime = stats.PowerAcPeakAllTime
		pv.PowerAcPeakToday = stats.PowerAcPeakToday
		pv.PowerAcPeakTodayTime = stats.PowerAcPeakTodayTime
		if !firstRun {
			pvCh := make(chan PvData)
			reqCh <- pvCh
			pv = <-pvCh
		}
		pv, err := updateFast(initiateData, pv)
		if firstRun {
			pv, err = updateSlow(initiateData, pv)
		}
		if err != nil {
			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s", err.Error(), errCounter, plantkey)
		} else {
			updateCh <- pv
			firstRun = false
		}
		if errCounter > errClose {
			break LOOP
		}
		//Wait for rerequest
		log.Debug("Waiting on tickers...")
		select {
		case <-fastTickCh:
			// Restart terminate ticker
		case <-slowTickCh:
			pvCh := make(chan PvData)
			reqCh <- pvCh
			pv := <-pvCh
			pv, err := updateSlow(initiateData, pv)
			if err != nil {
				errCounter++
				log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s", err.Error(), errCounter, plantkey)
			} else {
				updateCh <- pv
			}
			
			saveStats(plantkey, &pv)
			if errCounter > errClose {
				break LOOP
			}

		case <-terminateCh:
			break LOOP
		}

	}

	shutdown()
}

func midnight() time.Time {
	y,m,d := time.Now().Date();
	return time.Date(y,m,d,0,0,0,0,time.Local)
}

// This will handle the latestdata we have
// Syncronized getter and setter of the data
// Terminate on any signal on terminateCh
// Should run as a goroutine
func LatestPvData(reqCh chan chan PvData,
	updateCh chan PvData,
	terminateCh chan int) {
	latestData := PvData{}

	// Wait here until first data is received 
/*
	select {
	case latestData = <-updateCh:
		latestData.LatestUpdate = time.Now()
	case <-terminateCh:
		log.Debug("Terminated latestData")
		return
	}
*/
	for {
		select {
		case latestData = <-updateCh:
			if latestData.PowerAcPeakAll < latestData.PowerAc {
				latestData.PowerAcPeakAll = latestData.PowerAc
				latestData.PowerAcPeakAllTime = time.Now();
			}
			
			// Reset peak today if lasttime is less than now
			if latestData.PowerAcPeakTodayTime.Before(midnight()) {
				log.Debugf("Resetting PowerAcPeakToday because we passed midnight, %s is before %s",latestData.PowerAcPeakTodayTime, midnight())
				latestData.PowerAcPeakTodayTime = time.Now()
				latestData.PowerAcPeakToday = latestData.PowerAc
			}
			if latestData.PowerAcPeakToday < latestData.PowerAc {
				log.Debugf("Updating PowerAcPeakToday beause the latest (%i) was higher", latestData.PowerAc)
				latestData.PowerAcPeakToday = latestData.PowerAc
				latestData.PowerAcPeakTodayTime = time.Now();
			}			
			t := time.Now()
			latestData.LatestUpdate = &t 
		case repCh := <-reqCh:
			repCh <- latestData
		case <-terminateCh:
			log.Debug("Terminated latestData")
			return
		}
	}
}
