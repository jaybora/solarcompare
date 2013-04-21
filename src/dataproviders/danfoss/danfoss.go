package danfoss

import (
	"dataproviders"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"net/url"
	"time"
	"regexp"
	"strconv"
)


type dataProvider struct {
	InitiateData   dataproviders.InitiateData
	latestErr      error
	client         *http.Client
}


var log = logger.NewLogger(logger.DEBUG, "Dataprovider: Danfoss:")

const MAX_ERRORS = 5

const urlTemplate = "http://%s/%s"
const loginUrl = "cgi-bin/handle_login.tcl"
const logoutUrl = "cgi-bin/logout.tcl?sid=%s"
const forceLogoutUrl = "cgi-bin/closed_for_now.tcl?useTheForce=1"
const pacUrl = "cgi-bin/overview.tcl?sid=%s"

const sidRegEx = "sid=[0-9]+"
const curPwr1RegEx = "<td id=\"curr_power\" class=\"parValue\" style=\"width:25%;\">[0-9|\\.]+ [kW|W]"
const numberRegEx = ">[0-9|\\.]+"
const etodayRegEx = "<td id=\"prod_today\" class=\"parValue\">[0-9|\\.]+ [kW|W]"
const etotalRegEx = "<td id=\"total_yield\" class=\"parValue\">[0-9|\\.]+"
func (d *dataProvider) Name() string {
	return "N/A"
}

func NewDataProvider(initiateData dataproviders.InitiateData, 
                     term dataproviders.TerminateCallback,
                     client *http.Client,
                     pvStore dataproviders.PvStore,
                     statsStore dataproviders.PlantStatsStore) dataProvider {
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
		time.Second * 10,
		time.Minute * 5,
		time.Minute * 30,
		term,
		MAX_ERRORS,
		statsStore,
		pvStore)

	return dp
}

// Update PvData
func updatePvData(client *http.Client, initiateData *dataproviders.InitiateData, pv *dataproviders.PvData) error {
	log.Debug("Fetching update ...")
	sid, err := login(client, initiateData)
	if err != nil {
		// Try force logout
		forcelogout(client, initiateData)
		return err
	}
	
	b, err := genericdata(sid, client, initiateData)
	
	err = pac(sid, client, initiateData, pv, &b)
	if err != nil {return err}
	err = etoday(sid, client, initiateData, pv, &b)
	if err != nil {return err}
	err = etotal(sid, client, initiateData, pv, &b)
	if err != nil {return err}
	logout(sid, client, initiateData)
	return nil
}

func login(client *http.Client, 
            initiateData *dataproviders.InitiateData, ) (sid string, err error) {
	//Do login
	loginurl := fmt.Sprintf(urlTemplate, initiateData.Address, loginUrl)
	postdata := url.Values{}
	postdata.Add("user", initiateData.UserName)
	postdata.Add("pw", initiateData.Password)
	postdata.Add("submit", "Log on")
	postdata.Add("sid", "")
	resp, err := client.PostForm(loginurl, postdata)
	if err != nil {
		log.Infof("Error in fetching from %s:%s", loginurl, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider danfoss fail. Received http status %d from server", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	log.Tracef("Received body from server: %s", b)

	reg, err := regexp.Compile(sidRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	found := reg.Find(b)
	if len(found) < 6 {
		err = fmt.Errorf("Could not login to inverter. Wrong username/password")
		return
	}
	sid = string(found[4:])
	log.Debugf("Login success. Sid is %s", sid)

    return  
}

func forcelogout(client *http.Client, 
            initiateData *dataproviders.InitiateData) {
	
	//Do logout 
	logouturl := fmt.Sprintf(urlTemplate, initiateData.Address, forceLogoutUrl)
	log.Debugf("Force logging out from inverter... url is %s", logouturl)
	resp, err := client.Get(logouturl)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider danfoss fail. Received http status %d from inverter doing force logout", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	log.Tracef("Received body from server: %s", b)
	resp.Body.Close()
}

func genericdata(sid string, client *http.Client, 
            initiateData *dataproviders.InitiateData) (b []byte, err error) {
	//Get data ----------------------------------------------------------------------------
	pacurl := fmt.Sprintf(urlTemplate, initiateData.Address, fmt.Sprintf(pacUrl, sid))
	log.Tracef("Getting data from inverter... url is %s", pacurl)
	resp, err := client.Get(pacurl)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider danfoss fail. Received http status %d from inverter doing data gathering", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return
}

func pac(sid string, client *http.Client, 
            initiateData *dataproviders.InitiateData, pv *dataproviders.PvData, resp *[]byte) error {
	
	reg, err := regexp.Compile(curPwr1RegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	foundPart1 := reg.Find(*resp)
	if len(foundPart1) < 6 {
		err = fmt.Errorf("Could not find curpower in response from inverter")
		return err
	}
	log.Debugf("Found part 1 to be '%s'", foundPart1)
	
	
	reg, err = regexp.Compile(numberRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	foundPart2 := reg.Find(foundPart1)
	if len(foundPart2) < 2 {
		err = fmt.Errorf("Could not find curpower in response from inverter")
		return err
	}

	pac := string(foundPart2[1:])
	log.Debugf("Current Pac is %s", pac)
	pacfloat, err := strconv.ParseFloat(pac, 64)
	
	// Are the value in kW or W?
	factor := 1.0
	if string(foundPart1[len(foundPart1)-1:len(foundPart1)]) == "k" {
		factor = 1000.0
	}
	pv.PowerAc = uint16(pacfloat*factor)
	
	return nil
}

func etoday(sid string, client *http.Client, 
            initiateData *dataproviders.InitiateData, pv *dataproviders.PvData, resp *[]byte) error {
	
	reg, err := regexp.Compile(etodayRegEx)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	foundPart1 := reg.Find(*resp)
	if len(foundPart1) < 6 {
		err := fmt.Errorf("Could not find etoday in response from inverter")
		return err
	}
	
	reg, err = regexp.Compile(numberRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	foundPart2 := reg.Find(foundPart1)
	if len(foundPart2) < 2 {
		err = fmt.Errorf("Could not find etoday in response from inverter")
		return err
	}

	etoday := string(foundPart2[1:])
	log.Debugf("Current etoday is %s", etoday)
	etodayfloat, err := strconv.ParseFloat(etoday, 64)
	
	// Are the value in kW or W?
	factor := 1.0
	if string(foundPart1[len(foundPart1)-1:len(foundPart1)]) == "k" {
		factor = 1000.0
	}
	pv.EnergyToday = uint16(etodayfloat*factor)
	return nil
}
func etotal(sid string, client *http.Client, 
            initiateData *dataproviders.InitiateData, pv *dataproviders.PvData, resp *[]byte) error {
	
	reg, err := regexp.Compile(etotalRegEx)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	foundPart1 := reg.Find(*resp)
	if len(foundPart1) < 6 {
		err := fmt.Errorf("Could not find etotal in response from inverter")
		return err
	}
	
	reg, err = regexp.Compile(numberRegEx)
	if err != nil {
		log.Fail(err.Error())
	}
	foundPart2 := reg.Find(foundPart1)
	if len(foundPart2) < 2 {
		err = fmt.Errorf("Could not find etotal in response from inverter")
		return err
	}

	etotal := string(foundPart2[1:])
	log.Debugf("Current etotal is %s", etotal)
	etotalfloat, err := strconv.ParseFloat(etotal, 64)
	
	pv.EnergyTotal = float32(etotalfloat)
	return nil
}

func logout(sid string, client *http.Client, initiateData *dataproviders.InitiateData) {
	//Do logout 
	logouturl := fmt.Sprintf(urlTemplate, initiateData.Address, fmt.Sprintf(logoutUrl, sid))
	log.Tracef("Logging out from inverter... url is %s", logouturl)
	resp, err := client.Get(logouturl)
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Dataprovider danfoss fail. Received http status %d from inverter doing logout", resp.StatusCode)
		log.Infof("%s", err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	log.Tracef("Received body from server: %s", b)
	resp.Body.Close()
}
