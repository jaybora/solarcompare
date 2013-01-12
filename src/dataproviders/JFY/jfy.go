package JFY

import (
	"dataproviders"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"time"
)

type jfyDataProvider struct {
	InitiateData   dataproviders.InitiateData
	latestReqCh    chan chan dataproviders.PvData
	latestUpdateCh chan dataproviders.PvData
	terminateCh    chan int
	latestErr      error
	client         *http.Client
}

const GetUrl = "http://cts.jbr.dk:81/json"

var log = logger.NewLogger(logger.TRACE, "Dataprovider: JFY:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 30 //secs

// This will handle the latestdata we have
// Syncronized getter and setter of the data
func latestData(reqCh chan chan dataproviders.PvData, 
                updateCh chan dataproviders.PvData,
                terminateCh chan int) {
	// Wait here until first data is received 
	latestData := <-updateCh

	for {
		select {
		case latestData = <-updateCh:
		case repCh := <-reqCh:
			repCh <- latestData
		case <- terminateCh:
			log.Debug("Terminated latestData")
			return
		}
	}
}

func (jfy *jfyDataProvider) Name() string {
	return "JFY"
}

func NewDataProvider(initiateData dataproviders.InitiateData, term dataproviders.TerminateCallback) jfyDataProvider {
	log.Debug("New JFY dataprovider")
	client := &http.Client{}

	jfy := jfyDataProvider{initiateData,
		make(chan chan dataproviders.PvData),
		make(chan dataproviders.PvData),
		make(chan int),
		nil,
		client}
	go RunUpdates(jfy.client, jfy.latestUpdateCh, term, jfy.terminateCh)
	go latestData(jfy.latestReqCh, jfy.latestUpdateCh, jfy.terminateCh)

	return jfy
}

func RunUpdates(client *http.Client, updateCh chan dataproviders.PvData, term dataproviders.TerminateCallback, termCh chan int) {
	// Refresh ticker
	ticker := time.NewTicker(time.Second * 5)
	tickerCh := ticker.C
	// Terminate ticker
	terminateTicker := time.NewTicker(time.Second * INACTIVE_TIMOUT)
	terminateCh := terminateTicker.C

	errCounter := 0

	for {
		pv, err := updatePvData(client)
		if err != nil {

			errCounter++
			log.Infof("There was on error on updatePvData: %s, error counter is now %d", err.Error(), errCounter)
		} else {
			updateCh <- pv
		}
		if errCounter > MAX_ERRORS {
			break
		}
		//Wait for rerequest
		log.Debug("Waiting on tickers...")
		select {
		case <-tickerCh:
			// Restart terminate ticker
		case <-terminateCh:
			log.Debug("Terminate ticker")
			ticker.Stop()
			terminateTicker.Stop()
			term()
			termCh <- 0
			log.Info("RunUpdates exited")
			return
		}

	}
	ticker.Stop()
	terminateTicker.Stop()
	term()
	termCh <- 0
	log.Info("RunUpdates exited")
}

// Update PvData
func updatePvData(client *http.Client) (pv dataproviders.PvData, err error) {
	log.Debug("Fetching update ...")
	resp, err := client.Get(GetUrl)
	if err != nil {
		log.Infof("Error in fetching from %s:%s", GetUrl, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider JFY fail. Received http status %d from server", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	log.Tracef("Received body from server: %s", b)
	err = json.Unmarshal(b, &pv)

	log.Tracef("pv is now %s", pv)

	return
}

// Get latest PvData
func (jfy *jfyDataProvider) PvData() (pv dataproviders.PvData, err error) {
	reqCh := make(chan dataproviders.PvData)
	jfy.latestReqCh <- reqCh
	pv = <-reqCh
	log.Tracef("Returning PvData as %s", pv)
	return
}
