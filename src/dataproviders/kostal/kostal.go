package kostal

import (
	"dataproviders"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type dataProvider struct {
	InitiateData dataproviders.InitiateData
	latestErr    error
	client       *http.Client
}

var log = logger.NewLogger(logger.INFO, "Dataprovider: Kostal:")

const MAX_ERRORS = 5

const urlTemplate = "http://%s"

const allValuesRegEx = "[0-9|\\.]+</td>"
const numberRegEx = "[0-9|\\.]+"

func (d *dataProvider) Name() string {
	return "N/A"
}

func NewDataProvider(initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback,
	client *http.Client,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore,
	terminateCh chan int) dataProvider {
	log.Debug("New dataprovider")

	dp := dataProvider{initiateData,
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
		time.Second*10,
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
func updatePvData(client *http.Client, initiateData *dataproviders.InitiateData, pv *dataproviders.PvData) error {
	log.Debug("Fetching update ...")

	b, err := genericdata(client, initiateData)
	values, err := parseToValues(&b)
	if err != nil {
		return err
	}

	// Pac
	pacFloat, err := parseToValue(values[0])
	if err != nil {
		return err
	}
	pv.PowerAc = uint16(pacFloat)

	//Energy total
	etFloat, err := parseToValue(values[1])
	if err != nil {
		return err
	}
	pv.EnergyTotal = float32(etFloat)

	// Energy today
	edFloat, err := parseToValue(values[2])
	if err != nil {
		return err
	}
	pv.EnergyToday = uint16(edFloat * 1000)

	// Volt DC
	voltFloatSum, err := parseToValue(values[3])
	if err != nil {
		return err
	}
	voltFloat, err := parseToValue(values[7])
	if err != nil {
		return err
	}
	voltFloatSum += voltFloat
	voltFloat, err = parseToValue(values[11])
	if err != nil {
		return err
	}
	voltFloatSum += voltFloat
	pv.VoltDc = float32(voltFloatSum)

	//Amp DC
	ampFloatSum, err := parseToValue(values[5])
	if err != nil {
		return err
	}
	ampFloat, err := parseToValue(values[9])
	if err != nil {
		return err
	}
	ampFloatSum += ampFloat
	ampFloat, err = parseToValue(values[13])
	if err != nil {
		return err
	}
	ampFloatSum += ampFloat
	pv.AmpereAc = float32(ampFloatSum)

	//	err = pac(sid, client, initiateData, pv, &b)
	//	err = etoday(sid, client, initiateData, pv, &b)
	//	if err != nil {return err}
	//	err = etotal(sid, client, initiateData, pv, &b)
	//	if err != nil {return err}
	//	logout(sid, client, initiateData)
	return nil
}

func genericdata(client *http.Client,
	initiateData *dataproviders.InitiateData) (b []byte, err error) {
	//Get data ----------------------------------------------------------------------------
	url := fmt.Sprintf(urlTemplate, initiateData.Address)
	log.Tracef("Getting data from inverter... url is %s", url)
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(initiateData.UserName, initiateData.Password)
	resp, err := client.Do(req)

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider kostal fail. Received http status %d from inverter doing data gathering", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)
	log.Debugf("Recieved %s", b)
	resp.Body.Close()
	return
}

func parseToValues(resp *[]byte) (values [][]byte, err error) {

	reg, err := regexp.Compile(allValuesRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	values = reg.FindAll(*resp, -1)
	if len(values) < 1 {
		err = fmt.Errorf("Could not find any values in response from inverter")

	}
	log.Debugf("Found values to be '%s'", values)

	return
}

func parseToValue(unparsedValue []byte) (value float64, err error) {

	reg, err := regexp.Compile(numberRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	valstr := reg.Find(unparsedValue)
	if len(valstr) < 1 {
		err = fmt.Errorf("Could not find value in response from inverter")
		return
	}

	log.Debugf("Value as string is %s", valstr)
	value, err = strconv.ParseFloat(string(valstr), 64)
	return
}

//
//func etoday(sid string, client *http.Client,
//            initiateData *dataproviders.InitiateData, pv *dataproviders.PvData, resp *[]byte) error {
//
//	reg, err := regexp.Compile(etodayRegEx)
//	if err != nil {
//		log.Fail(err.Error())f
//		return err
//	}
//	foundPart1 := reg.Find(*resp)
//	if len(foundPart1) < 6 {
//		err := fmt.Errorf("Could not find etoday in response from inverter")
//		return err
//	}
//
//	reg, err = regexp.Compile(numberRegEx)
//	if err != nil {
//		log.Fail(err.Error())
//	}
//	foundPart2 := reg.Find(foundPart1)
//	if len(foundPart2) < 2 {
//		err = fmt.Errorf("Could not find etoday in response from inverter")
//		return err
//	}
//
//	etoday := string(foundPart2[1:])
//	log.Debugf("Current etoday is %s", etoday)
//	etodayfloat, err := strconv.ParseFloat(etoday, 64)
//
//	// Are the value in kW or W?
//	factor := 1.0
//	if string(foundPart1[len(foundPart1)-1:len(foundPart1)]) == "k" {
//		factor = 1000.0
//	}
//	pv.EnergyToday = uint16(etodayfloat*factor)
//	return nil
//}
//func etotal(sid string, client *http.Client,
//            initiateData *dataproviders.InitiateData, pv *dataproviders.PvData, resp *[]byte) error {
//
//	reg, err := regexp.Compile(etotalRegEx)
//	if err != nil {
//		log.Fail(err.Error())
//		return err
//	}
//	foundPart1 := reg.Find(*resp)
//	if len(foundPart1) < 6 {
//		err := fmt.Errorf("Could not find etotal in response from inverter")
//		return err
//	}
//
//	reg, err = regexp.Compile(numberRegEx)
//	if err != nil {
//		log.Fail(err.Error())
//	}
//	foundPart2 := reg.Find(foundPart1)
//	if len(foundPart2) < 2 {
//		err = fmt.Errorf("Could not find etotal in response from inverter")
//		return err
//	}
//
//	etotal := string(foundPart2[1:])
//	log.Debugf("Current etotal is %s", etotal)
//	etotalfloat, err := strconv.ParseFloat(etotal, 64)
//
//	pv.EnergyTotal = float32(etotalfloat)
//	return nil
//}
//
//func logout(sid string, client *http.Client, initiateData *dataproviders.InitiateData) {
//	//Do logout
//	logouturl := fmt.Sprintf(urlTemplate, initiateData.Address, fmt.Sprintf(logoutUrl, sid))
//	log.Tracef("Logging out from inverter... url is %s", logouturl)
//	resp, err := client.Get(logouturl)
//	if resp.StatusCode != 200 {
//		err = fmt.Errorf("Dataprovider danfoss fail. Received http status %d from inverter doing logout", resp.StatusCode)
//		log.Infof("%s", err.Error())
//		return
//	}
//	defer resp.Body.Close()
//	b, _ := ioutil.ReadAll(resp.Body)
//	log.Tracef("Received body from server: %s", b)
//	resp.Body.Close()
//}
