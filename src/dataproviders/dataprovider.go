package dataproviders

import (
	"logger"
	"time"
)

type InitiateData struct {
	UserName string
	Password string
	PlantNo  string
}

var log = logger.NewLogger(logger.TRACE, "Dataprovider: generic: ")

type DataProvider interface {
	Name() string
	PvData() (pv PvData, err error)
}

type TerminateCallback func()

type UpdatePvData func(pv PvData) (pv PvData, err error)

// List of all known dataproviders
// This would be nice if we could autodetect them somehow
const (
	FJY         = iota
	SunnyPortal = iota
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
func RunUpdates(updateFast UpdatePvData,
	updateSlow UpdatePvData,
	fastTime time.Duration,
	slowTime time.Duration,
	terminateTime time.Duration,
	updateCh chan PvData,
	reqCh chan chan PvData,
	term TerminateCallback,
	termCh chan int,
	errClose int) {
	log.Trace("Started a RunUpdates rutine")

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

LOOP:
	for {
		// If this is first run, then reqCh will block
		// So we start with a zero'ed pv
		pv := PvData{}
		if !firstRun {
			pvCh := make(chan PvData)
			reqCh <- pvCh
			pv = <-pvCh
		}
		pv, err := updateFast(pv)
		if firstRun {
			pv, err = updateSlow(pv)
		}
		if err != nil {
			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d", err.Error(), errCounter)
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
			pv, err := updateSlow(pv)
			if err != nil {
				errCounter++
				log.Infof("There was on error on updatePvData: %s, error counter is now %d", err.Error(), errCounter)
			} else {
				updateCh <- pv
			}
			if errCounter > errClose {
				break LOOP
			}

		case <-terminateCh:
			break LOOP
		}

	}
	log.Debug("Terminate ticker")
	fastTick.Stop()
	slowTick.Stop()
	terminateTicker.Stop()
	term()
	termCh <- 0
	log.Info("RunUpdates exited")
}

// This will handle the latestdata we have
// Syncronized getter and setter of the data
// Terminate on any signal on terminateCh
// Should run as a goroutine
func LatestPvData(reqCh chan chan PvData,
	updateCh chan PvData,
	terminateCh chan int) {
	// Wait here until first data is received 
	latestData := <-updateCh

	for {
		select {
		case latestData = <-updateCh:
		case repCh := <-reqCh:
			repCh <- latestData
		case <-terminateCh:
			log.Debug("Terminated latestData")
			return
		}
	}
}
