package solaredge

import (
	"dataproviders"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* JSON GET output from
   http://monitoring.solaredge.com/solaredge-web/p/public_dashboard_data?fieldId=22175:

{
"sumData":{
            "siteName":"Moelagerhus",

  "siteInstallationDate":"11/22/2012",
  "sitePeakPower":"7.5",
    "siteCountry":"Denmark",
    "siteUpdateDate":"08/03/2013 21:36",
      "maxSeverity":"0",
    "status":"Active"
},
"instData":{
  "installerImage":"",
  "installerImagePath":"",
  "installerImageHash":""
},
"imgData":{
  "fieldId":"22175",
  "image":"PICT0168.JPG",
  "imagePath":"/public_fieldImage?fieldId=22175",
  "imageHash":"1480262490"
},
"overviewData":{
    "lastDayEnergy":"25.21 kWh",
      "lastMonthEnergy":"76.48 kWh",
      "currentPower":"0 W",
      "lifeTimeEnergy":"4.59 MWh",
      "lifeTimeRevenue":"kr10,361.96"
},
"savingData":{
    "CO2EmissionSaved":"1,799.11 kg",
    "treesEquivalentSaved":"6.01",
  "powerSavedInLightBulbs":"13,907.77"
},
"powerChartData":{
  "start_week":1374969600000,
  "energy_chart_month":[
  {"name":2012,"data":[0,0,0,0,0,0,0,0,0,0,0.01,0.052]}
  ,{"name":2013,"data":[0.091,0.136,0.52,0.725,0.919,0.985,1.075,0.076,0,0,0,0]}
  ],
  "energy_chart_month_max":1.075,
  "month_uom":"mega",
  "energy_chart_quarter":[
  {"name":2012,"data":[0,0,0,0.062]}
  ,{"name":2013,"data":[0.747,2.629,1.151,0]}
  ],
  "quarter_uom":"mega",
  "year_categories":[2012, 2013],
  "energy_chart_year":[
  {"name":2012,"data":[0.063,0]}
  ,{"name":2013,"data":[0,4.527]}
  ],
  "year_uom":"mega"
},
"energyChartData":{
  "field_start_date":{"year":2012,"month":10,"day":22},
  "field_end_date":{"year":2013,"month":7,"day":3},
    "year_range":[[2012],[2013]],
  	start_week:1374969600000,
start_week_obj:{"year":2013,"month":6,"day":28},
end_week:{"year":2013,"month":7,"day":3},
end_week_next:1375574340000,
power_chart_week:[null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0.009,0.089,0.258,0.458,0.416,0.386,0.781,1.349,1.349,0.684,0.67,1.697,0.743,0.819,0.925,1.402,2.064,2.048,1.959,3.334,2.855,3.841,3.884,3.089,3.123,3.767,3.722,4.128,4.006,3.119,4.361,4.331,4.322,4.193,4.105,4.207,4.353,4.373,2.812,4.211,4.123,3.81,3.701,3.516,3.359,3.117,2.991,2.971,1.713,1.257,1.099,1.222,0.754,0.681,0.532,0.298,0.182,0.016,0,0,0,0,0,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0.004,0.066,0.127,0.341,0.647,0.59,0.559,0.783,0.905,0.833,1.229,1.397,1.685,1.297,1.926,2.117,3.297,2.29,2.558,3.407,4.161,3.398,3.494,3.702,3.66,3.693,2.646,2.33,2.69,2.458,2.411,2.331,2.903,2.22,2.052,2.362,2.433,2.112,2.299,1.843,1.557,1.911,2.017,1.795,2.35,3.62,3.408,3.437,2.609,2.321,2.1,1.848,1.412,0.92,0.688,0.5,0.23,0.246,0.171,0.161,0.073,0,0,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0.002,0.085,0.133,0.159,0.195,0.334,0.331,0.606,0.881,0.84,1.525,3.078,1.457,1.41,2.01,1.8,1.99,2.616,4.014,1.885,2.07,4.227,4.171,4.58,3.907,4.579,3.866,3.643,4.619,4.706,4.695,4.976,2.527,4.396,4.333,3.838,4.36,4.375,3.622,3.766,2.888,3.917,2.689,2.914,3.066,2.751,1.595,2.123,2.126,1.818,1.52,0.961,0.833,0.512,0.369,0.343,0.354,0.231,0.118,0.037,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0.003,0.169,0.121,0.168,0.21,0.335,0.64,0.518,0.573,0.83,1.836,2.304,2.396,2.429,2.601,3.474,2.653,3.75,2.97,3.745,4.3,3.065,2.019,3.923,4.895,4.164,2.291,2.724,3.65,3.236,5.393,4.53,2.777,0.406,2.832,3.024,4.56,3.67,4.05,3.949,3.904,3.128,3.039,1.864,2.647,1.196,1.333,1.631,1.876,1.422,1.74,1.199,0.879,0.691,0.317,0.216,0.167,0.133,0.084,0.005,0,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0.076,0.137,0.238,0.119,0.295,0.409,0.482,0.589,1.279,1.321,1.447,1.111,1.511,1.588,1.59,1.575,1.597,1.803,1.59,1.472,1.71,2.509,1.443,1.998,2.752,2.771,1.492,1.213,1.43,1.155,0.657,0.303,1.034,0.365,0.933,1.192,1.411,1.092,0.8,0.494,0.916,0.747,1.045,0.604,0.396,0.807,0.545,0.368,0.26,0.34,0.643,0.984,1.01,1.119,0.89,0.54,0.523,0.279,0.122,0.027,0.077,0.054,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0,0.041,0.221,0.432,0.583,0.705,0.835,0.994,1.181,1.36,1.496,1.663,1.872,1.995,2.21,2.422,2.663,2.763,2.828,3.073,3.209,3.505,3.622,3.708,3.786,3.859,4.066,4.116,4.138,4.149,4.177,4.191,4.164,4.149,4.132,4.09,4.041,3.964,3.911,3.819,3.716,3.607,3.48,2.413,3.257,3.038,2.856,2.653,2.462,2.247,2.034,1.74,1.436,1.081,0.724,0.419,0.254,0.197,0.161,0.121,0.067,0.003,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,0,0,0.001,0.072,0.208,0.368,0.517,0.646,0.785,0.964,1.13,1.298,1.41,1.637,1.711,1.936,1.038,1.733,1.792,2.074,2.225,1.948,3.236,3.206,2.998,3.014,3.627,3.794,3.831,1.823,2.507,2.488,1.988,1.948,2.752,2.679,2.038,1.228,0.388,0.31,0.28,2.237,2.029,2.103,2.656,2.823,2.406,3.074,2.668,1.736,2.343,2.691,2.119,1.69,0.592,0.872,0.567,0.438,0.281,0.328,0.173,0,0.025,0.02,0,0,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null],
power_chart_week_c:[],
"energy_chart_month_by_day":[
{"name":"Production: 76.484 KWh","data":[14.781,36.494,25.209,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"color":"#40b463"}
],
"energy_chart_month_by_day_total":76.484,
"energy_chart_month_by_day_max":36.494,
"energy_chart_month_by_day_total_c":0,
"energy_chart_month_by_day_max_c":0,
"energy_chart_year_by_month":[
{"name":"Production: 4.527 MWh","data":[0.091,0.136,0.52,0.725,0.919,0.985,1.075,0.076,0,0,0,0],"color":"#40b463"}
],
"energy_chart_year_by_month_total":4.527,
"energy_chart_year_by_month_max":1.075,
"energy_chart_year_by_month_total_c":0,
"energy_chart_year_by_month_max_c":0,
"prev_month_date":1375228800000,
"next_month_date":1375574399000,
"prev_year_date":1356912000000,
"next_year_date":1375574399000,
"year_uom":"mega",
"month_uom":"kilo" }
}
*/

type dataProvider struct {
	InitiateData dataproviders.InitiateData
	//	latestReqCh    chan chan dataproviders.PvData
	//	latestUpdateCh chan dataproviders.PvData
	//	terminateCh    chan int
	latestErr error
	client    *http.Client
}

type overviewData struct {
	LastDayEnergy  string `json:"lastDayEnergy"`
	CurrentPower   string `json:"currentPower"`
	LifeTimeEnergy string `json:"lifeTimeEnergy"`
}

type jsonData struct {
	OverviewData overviewData `json:"overviewData"`
}

const dataUrl = "http://monitoring.solaredge.com/solaredge-web/p/public_dashboard_data?fieldId=%s"

var log = logger.NewLogger(logger.INFO, "Dataprovider: solaredge:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 30 //secs

func (dp *dataProvider) Name() string {
	return "Solaredge"
}

func NewDataProvider(initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback,
	client *http.Client,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore,
	terminateCh chan int) dataProvider {
	log.Debug("New dataprovider")

	dp := dataProvider{initiateData,
		//		make(chan chan dataproviders.PvData),
		//		make(chan dataproviders.PvData),
		//		make(chan int),
		nil,
		client}
	go dataproviders.RunUpdates(
		&initiateData,
		func(initiateData *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			err := updatePvData(client, initiateData, pv)
			pv.LatestUpdate = nil
			return err
		},
		func(initiateData *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			return nil
		},
		time.Second*60,
		time.Minute*5,
		time.Minute*30,
		terminateCh,
		term,
		MAX_ERRORS,
		statsStore,
		pvStore)

	return dp
}

// Update PvData
func updatePvData(client *http.Client,
	initiateData *dataproviders.InitiateData,
	pv *dataproviders.PvData) error {

	log.Debug("Fetching update ...")
	url := fmt.Sprintf(dataUrl, initiateData.PlantNo)
	resp, err := client.Get(url)
	if err != nil {
		log.Infof("Error in fetching from %s:%s", url, err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider suntrol fail. Received http status %d from server", resp.StatusCode)
		log.Infof("%s", err.Error())
		return err
	}
	b, _ := ioutil.ReadAll(resp.Body)

	log.Tracef("Received body from server: %s", b)
	// There is an error in the json from the server so we filter it with regexp
	reg, err := regexp.Compile("\"overviewData(?s)(.*?)}")
	if err != nil {
		log.Fail(err.Error())
	}
	found := "{" + string(reg.Find(b)) + "}"
	log.Tracef("After filtering to overviewData only: %s", found)

	jd := jsonData{}
	err = json.Unmarshal([]byte(found), &jd)
	if err != nil {
		log.Infof("Error in umashalling json %s", err.Error())
		return err
	}

	log.Tracef("Unmashaled overviewdata is %s", jd)

	pv.PowerAc = uint16(calcUnitFactor(jd.OverviewData.CurrentPower))

	pv.EnergyToday = uint16(calcUnitFactor(jd.OverviewData.LastDayEnergy))
	pv.EnergyTotal = float32(float32(calcUnitFactor(jd.OverviewData.LifeTimeEnergy) / 1000))

	//pv.EnergyToday = uint16(chartData.DataPart[time.Now().Day()-1].Value * 1000)

	log.Tracef("pv is now %s", pv)

	return nil
}

func calcUnitFactor(data string) uint {
	values := strings.Split(data, " ")

	value, _ := strconv.ParseFloat(values[0], 64)
	unit := values[1]

	factor := 0.0

	switch unit[0] {
	case 'W':
		factor = 1
	case 'k':
		factor = 1000
	case 'M':
		factor = 1000000
	}

	return uint(value * factor)

}
