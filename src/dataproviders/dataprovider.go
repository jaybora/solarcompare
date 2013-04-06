package dataproviders

import (
	"logger"
	"time"
)

type PlantStats struct {
	PowerAcPeakAll       uint16
	PowerAcPeakAllTime   time.Time
	PowerAcPeakToday     uint16
	PowerAcPeakTodayTime time.Time
}

type PlantStatsStore interface {
	LoadStats(plantkey string) PlantStats
	SaveStats(plantkey string, pv *PvData)
}

type InitiateData struct {
	PlantKey string
	UserName string
	Password string
	PlantNo  string
	Address  string
}

var log = logger.NewLogger(logger.DEBUG, "Dataprovider: generic: ")

type DataProvider interface {
	Name() string
	PvData() (pv PvData, err error)
}

type TerminateCallback func()

// Callback event that is run whenever updated data is available
type PvDataUpdatedEvent func(plantkey string, pv PvData)

type UpdatePvData func(i *InitiateData, pv PvData) (pv PvData, err error)

// List of all known dataproviders
// This would be nice if we could autodetect them somehow
const (
	FJY         = iota
	SunnyPortal = iota
	Suntrol     = iota
	Danfoss     = iota
)

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
	statsStore PlantStatsStore) {

	log.Trace("Started a RunUpdates rutine")
	stats := statsStore.LoadStats(initiateData.PlantKey)

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
		log.Infof("About to terminate RunUpdates for plant %s", initiateData.PlantKey)
		fastTick.Stop()
		slowTick.Stop()
		terminateTicker.Stop()
		pvCh := make(chan PvData)
		reqCh <- pvCh
		pv := <-pvCh
		statsStore.SaveStats(initiateData.PlantKey, &pv)
		term()
		termCh <- 0
		log.Infof("RunUpdates exited for plant %s", initiateData.PlantKey)
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
		if err != nil {
			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
				err.Error(), errCounter, initiateData.PlantKey)
		}
		if firstRun {
			pv, err = updateSlow(initiateData, pv)
		}
		if err != nil {
			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
				err.Error(), errCounter, initiateData.PlantKey)
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
				log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
					err.Error(), errCounter, initiateData.PlantKey)
			} else {
				updateCh <- pv
			}

			statsStore.SaveStats(initiateData.PlantKey, &pv)
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
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// This will handle the latestdata we have
// Syncronized getter and setter of the data
// Terminate on any signal on terminateCh
// Should run as a goroutine
func LatestPvData(reqCh chan chan PvData,
	updateCh chan PvData,
	terminateCh chan int,
	pvDataUpdatedEvent PvDataUpdatedEvent,
	plantKey string) {
	latestData := PvData{}

	for {
		select {
		case newLatestData := <-updateCh:
			if newLatestData.LatestUpdate == nil {
				t := time.Now()
				newLatestData.LatestUpdate = &t
			}

			latestData.PowerAc = newLatestData.PowerAc
			latestData.AmpereAc = newLatestData.AmpereAc
			latestData.EnergyToday = newLatestData.EnergyToday
			latestData.EnergyTotal = newLatestData.EnergyTotal
			latestData.State = newLatestData.State
			latestData.VoltDc = newLatestData.VoltDc
			latestData.LatestUpdate = newLatestData.LatestUpdate
			
			// If the Peak is set in the new update use that
			if newLatestData.PowerAcPeakAll > latestData.PowerAcPeakAll {
				latestData.PowerAcPeakAll = newLatestData.PowerAcPeakAll
				latestData.PowerAcPeakAllTime = newLatestData.PowerAcPeakAllTime	
			}
			if newLatestData.PowerAcPeakToday > latestData.PowerAcPeakToday {
				latestData.PowerAcPeakToday = newLatestData.PowerAcPeakToday
				latestData.PowerAcPeakTodayTime = newLatestData.PowerAcPeakTodayTime	
			}

			if latestData.PowerAcPeakAll < newLatestData.PowerAc {
				latestData.PowerAcPeakAll = newLatestData.PowerAc
				latestData.PowerAcPeakAllTime = *newLatestData.LatestUpdate
			}

			// Reset peak today if lasttime is less than now
			if latestData.PowerAcPeakTodayTime.Before(midnight()) {
				log.Debugf("Resetting PowerAcPeakToday because we passed midnight, %s is before %s", latestData.PowerAcPeakTodayTime, midnight())
				latestData.PowerAcPeakToday = newLatestData.PowerAc
				latestData.PowerAcPeakTodayTime = *newLatestData.LatestUpdate
			}
			if latestData.PowerAcPeakToday < newLatestData.PowerAc {
				log.Debugf("Updating PowerAcPeakToday beause the latest (%i) was higher", newLatestData.PowerAc)
				latestData.PowerAcPeakToday = newLatestData.PowerAc
				latestData.PowerAcPeakTodayTime = *newLatestData.LatestUpdate
			}
			pvDataUpdatedEvent(plantKey, latestData)
		case repCh := <-reqCh:
			repCh <- latestData
		case <-terminateCh:
			log.Debug("Terminated latestData")
			return
		}
	}
}
