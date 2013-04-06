package suntrol

import (
	"dataproviders"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"time"
)

type dataProvider struct {
	InitiateData   dataproviders.InitiateData
	latestReqCh    chan chan dataproviders.PvData
	latestUpdateCh chan dataproviders.PvData
	terminateCh    chan int
	latestErr      error
	client         *http.Client
}



type dataPart struct {
	Value float32 `json:"value"`
}

type chartData struct {
	DataPart []dataPart `json:"data"`
}

const requestDateFormat = "2006-01"


// 1st parameter is building pid / PlantNo
// 2nd parameter is yyyy-mm
const MonthUrl = "http://suntrol-portal.com/en/plant/graph-json/month/p/1/pid/%s/date/%s/size/page/chart/Column3D/axis/static/output/real"

var log = logger.NewLogger(logger.INFO, "Dataprovider: Suntrol:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 30 //secs

func (dp *dataProvider) Name() string {
	return "Suntrol"
}

func NewDataProvider(initiateData dataproviders.InitiateData, 
                     term dataproviders.TerminateCallback,
                     client *http.Client,
                     pvDataUpdatedEvent dataproviders.PvDataUpdatedEvent,
                     statsStore dataproviders.PlantStatsStore) dataProvider {
	log.Debug("New dataprovider")

	dp := dataProvider{initiateData,
		make(chan chan dataproviders.PvData),
		make(chan dataproviders.PvData),
		make(chan int),
		nil,
		client}
	go dataproviders.RunUpdates(
		&initiateData,
		func(initiateData *dataproviders.InitiateData, pvint dataproviders.PvData) (pv dataproviders.PvData, err error) {
			pv, err = updatePvData(client, initiateData)
			return
		}, 
		func(initiateData *dataproviders.InitiateData, pvint dataproviders.PvData) (pv dataproviders.PvData, err error) {
			// To prevent zero when provider is starting up
			pv = pvint
			return
		}, 
		time.Minute * 1,
		time.Minute * 5,
		time.Minute * 30,
		dp.latestUpdateCh,
		dp.latestReqCh,
		term,
		dp.terminateCh,
		MAX_ERRORS,
		statsStore)
	go dataproviders.LatestPvData(dp.latestReqCh, dp.latestUpdateCh, dp.terminateCh,
		pvDataUpdatedEvent, initiateData.PlantKey)

	return dp
}

// Update PvData
func updatePvData(client *http.Client, initiateData *dataproviders.InitiateData) (pv dataproviders.PvData, err error) {
	log.Debug("Fetching update ...")
	url := fmt.Sprintf(MonthUrl, initiateData.PlantNo, time.Now().Format(requestDateFormat))
	resp, err := client.Get(url)
	if err != nil {
		log.Infof("Error in fetching from %s:%s", url, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider suntrol fail. Received http status %d from server", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	log.Tracef("Received body from server: %s", b)
	
	
	
	chartData := chartData{}
	err = json.Unmarshal(b, &chartData)
	if err != nil {
		log.Infof("Error in umashalling json %s", err.Error())
		return
	}

	log.Tracef("Unmashaled charData is %s", chartData);	
	
	pv.EnergyToday = uint16(chartData.DataPart[time.Now().Day()-1].Value * 1000)

	log.Tracef("pv is now %s", pv)

	return
}

// Get latest PvData
func (dp *dataProvider) PvData() (pv dataproviders.PvData, err error) {
	reqCh := make(chan dataproviders.PvData)
	dp.latestReqCh <- reqCh
	pv = <-reqCh
	log.Tracef("Returning PvData as %s", pv)
	return
}
