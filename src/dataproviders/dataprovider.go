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

// Interface for setting and getting pvdata
type PvStore interface {
	Set(plantkey string, pv *PvData)
	Get(plantkey string) PvData
}

type InitiateData struct {
	PlantKey string
	UserName string
	Password string
	PlantNo  string
	Address  string
}

var log = logger.NewLogger(logger.DEBUG, "Dataprovider: generic: ")

type TerminateCallback func()

type UpdatePvData func(i *InitiateData, pv *PvData) error

// List of all known dataproviders
// This would be nice if we could autodetect them somehow
const (
	FJY         = iota
	SunnyPortal = iota
	Suntrol     = iota
	Danfoss     = iota
	Kostal      = iota
)

type DataProviderDescription struct {
	DataProvider   uint
	Name           string
	RequiredFields []string
}

var DataProviders = []DataProviderDescription{
	{FJY, "FJY", []string{}},
	{SunnyPortal, "SunnyPortal", []string{"UserName", "Password", "PlantNo"}},
	{Suntrol, "Suntrol", []string{"PlantNo"}},
	{Danfoss, "Danfoss", []string{"UserName", "Password", "Address"}},
	{Kostal, "Kostal", []string{"UserName", "Password", "Address"}}}

// RunUpdates on the provider.
// updateFast, a function that gets called when a fast update is scheduled
// updateSlow, a function that gets called when a slow update is scheduled
// fastTime, secs on updateFast should be scheduled
// slowTime, secs on updateSlow should be scheduled
// terminateTime, secs on how long the provider will stay online before it terminates
// terminateCh, signaling here will terminate the dataprovider
// term, a function that gets called when RunUpdates terminates
// errClose, maximum number of errors received before giving up, and terminates
// statsStore service for storinging peak
// pvStore store for setting and getting actual data
func RunUpdates(initiateData *InitiateData,
	updateFast UpdatePvData,
	updateSlow UpdatePvData,
	fastTime time.Duration,
	slowTime time.Duration,
	terminateTime time.Duration,
	terminateCh chan int,
	term TerminateCallback,
	errClose int,
	statsStore PlantStatsStore,
	pvStore PvStore) {

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
	terminateChTick := terminateTicker.C

	errCounter := 0
	firstRun := true

	shutdown := func() {
		log.Infof("About to terminate RunUpdates for plant %s", initiateData.PlantKey)
		fastTick.Stop()
		slowTick.Stop()
		terminateTicker.Stop()
		pv := pvStore.Get(initiateData.PlantKey)
		statsStore.SaveStats(initiateData.PlantKey, &pv)
		term()
		log.Infof("RunUpdates exited for plant %s", initiateData.PlantKey)
		return
	}

	defer func() {
		shutdown()
		if r := recover(); r != nil {
			log.Infof("Recovered in dataprovider.RunUpdates, %s", r)
		}
	}()

	// Loud up from previous pvdata, then setup peak from store
	pv := pvStore.Get(initiateData.PlantKey)
	pv.PowerAcPeakAll = stats.PowerAcPeakAll
	pv.PowerAcPeakAllTime = stats.PowerAcPeakAllTime
	pv.PowerAcPeakToday = stats.PowerAcPeakToday
	pv.PowerAcPeakTodayTime = stats.PowerAcPeakTodayTime
	pvStore.Set(initiateData.PlantKey, &pv)
LOOP:
	for {
		err := updateFast(initiateData, &pv)
		if err != nil {
			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
				err.Error(), errCounter, initiateData.PlantKey)
		} else {
			updatePvPeak(pvStore, &initiateData.PlantKey, &pv)
		}

		if firstRun {
			err = updateSlow(initiateData, &pv)
			if err != nil {
				errCounter++
				log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
					err.Error(), errCounter, initiateData.PlantKey)
			} else {
				updatePvPeak(pvStore, &initiateData.PlantKey, &pv)
			}
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
			err := updateSlow(initiateData, &pv)
			if err != nil {
				errCounter++
				log.Infof("There was on error on updatePvData: %s, error counter is now %d for plant %s",
					err.Error(), errCounter, initiateData.PlantKey)
			} else {
				updatePvPeak(pvStore, &initiateData.PlantKey, &pv)
			}

			statsStore.SaveStats(initiateData.PlantKey, &pv)
			if errCounter > errClose {
				break LOOP
			}

		case <-terminateCh:
			break LOOP

		case <-terminateChTick:
			break LOOP
		}
	}

	shutdown()
}

func midnight() time.Time {
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func updatePvPeak(pvStore PvStore, plantKey *string, pv *PvData) {
	if pv.LatestUpdate == nil {
		t := time.Now()
		pv.LatestUpdate = &t
	}
	if pv.PowerAcPeakAll < pv.PowerAc {
		pv.PowerAcPeakAll = pv.PowerAc
		pv.PowerAcPeakAllTime = *pv.LatestUpdate
	}

	// Reset peak today if lasttime is less than now
	if pv.PowerAcPeakTodayTime.Before(midnight()) {
		log.Debugf("Resetting PowerAcPeakToday because we passed midnight, %s is before %s", pv.PowerAcPeakTodayTime, midnight())
		pv.PowerAcPeakToday = pv.PowerAc
		pv.PowerAcPeakTodayTime = *pv.LatestUpdate
	}
	if pv.PowerAcPeakToday < pv.PowerAc {
		log.Debugf("Updating PowerAcPeakToday beause the latest (%i) was higher", pv.PowerAc)
		pv.PowerAcPeakToday = pv.PowerAc
		pv.PowerAcPeakTodayTime = *pv.LatestUpdate
	}

	pvStore.Set(*plantKey, pv)
}
