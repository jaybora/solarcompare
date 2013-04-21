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
//	latestReqCh    chan chan dataproviders.PvData
//	latestUpdateCh chan dataproviders.PvData
//	terminateCh    chan int
	latestErr      error
	client         *http.Client
}

const GetUrl = "http://cts.jbr.dk:81/json"

var log = logger.NewLogger(logger.INFO, "Dataprovider: JFY:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 30 //secs

func (jfy *jfyDataProvider) Name() string {
	return "JFY"
}

func NewDataProvider(initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback, 
	client *http.Client,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore) jfyDataProvider {
	log.Debug("New JFY dataprovider")
	

	jfy := jfyDataProvider{initiateData,
//		make(chan chan dataproviders.PvData),
//		make(chan dataproviders.PvData),
//		make(chan int),
		nil,
		client}
	go dataproviders.RunUpdates(
		&initiateData,
		func(id *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			newpv, err := updatePvData(client)
			if err != nil {return err}
			pv.PowerAc = newpv.PowerAc
			pv.AmpereAc = newpv.AmpereAc
			pv.EnergyToday = newpv.EnergyToday
			pv.EnergyTotal = newpv.EnergyTotal
			pv.VoltDc = newpv.VoltDc
			pv.LatestUpdate = nil

			return nil
		},
		func(id *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			return nil
		},
		time.Second*5,
		time.Minute*5,
		time.Minute*30,
		term,
		MAX_ERRORS,
		statsStore,
		pvStore)
	//go dataproviders.LatestPvData(jfy.latestReqCh, jfy.latestUpdateCh, jfy.terminateCh, 
	//                              pvDataUpdatedEvent, initiateData.PlantKey)

	return jfy
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
	pv.LatestUpdate = nil
	log.Tracef("pv is now %s", pv)
	return
}

//// Get latest PvData
//func (jfy *jfyDataProvider) PvData() (pv dataproviders.PvData, err error) {
//	reqCh := make(chan dataproviders.PvData)
//	jfy.latestReqCh <- reqCh
//	pv = <-reqCh
//	log.Tracef("Returning PvData as %s", pv)
//	return
//}
