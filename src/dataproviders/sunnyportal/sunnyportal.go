package sunnyportal

import (
	"bufio"
	"bytes"
	"dataproviders"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* Example of recieved JSON for update pac as changed to at 1. august 2013
   Note that when the inverter is in standby currentPlantPower is not transmitted

{"Timestamp":"\/Date(1375477009869)\/",
 "Info":{
 	"currentPlantTime":"Freitag, 2. August 2013 20:56",
 	"currentPlantPowerUnit":"W",
 	"currentPlantPower":"0",
 	"currentPlantPowerPointerAngle":0},
 "PV":null,
 "FeedIn":null,
 "GridConsumption":null,
 "DirectConsumption":null,
 "SelfConsumption":null,
 "TotalConsumption":null,
 "DirectConsumptionQuote":null,
 "SelfConsumptionQuote":null,
 "AutarkyQuote":null,
 "BatteryIn":null,
 "BatteryOut":null,
 "BatteryChargeStatus":null,
 "OperationHealth":{
 	"Ok":1,
 	"Warning":0,
 	"Error":0,
 	"Unknown":0},
 "BatteryStateOfHealth":null,
 "InfoMessages":[],
 "WarningMessages":[],
 "ErrorMessages":[]}

*/

type sunnyDataProvider struct {
	InitiateData dataproviders.InitiateData
	latestErr    error
	client       *http.Client
	viewstate    string //Something that sma portal uses, must be posted to login
	Plantname    string
}

type smaPacReplyInfo struct {
	CurrentPlantPower string `json:"currentPlantPower"`
	CurrentPlantTime  string `json:"currentPlantTime"`
}

type smaPacReply struct {
	Info smaPacReplyInfo
}

var log = logger.NewLogger(logger.INFO, "Dataprovider: SunnyPortal:")

const MAX_ERRORS = 5

//const INACTIVE_TIMOUT = 300 //secs

const startUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const loginUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const profileUrl = "http://www.sunnyportal.com/FixedPages/PlantProfile.aspx"
const pacUrl = "http://www.sunnyportal.com/Dashboard"
const csvPostUrl = "http://sunnyportal.com/FixedPages/EnergyAndPower.aspx"
const csvUrl = "http://sunnyportal.com/Templates/DownloadDiagram.aspx?down=diag"
const plantSelectUrl = "http://sunnyportal.com/FixedPages/Dashboard.aspx"

const keyDateFormat = "20060102"
const smaCsvDateFormat = "1/2/06"
const smaWebDateFormat = "1/2/2006"

func (sunny *sunnyDataProvider) Name() string {
	return sunny.Plantname
}

func NewDataProvider(initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback, client *http.Client,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore,
	terminateCh chan int) (sunny sunnyDataProvider, err error) {

	log.Debug("New dataprovider")

	sunny = sunnyDataProvider{initiateData,
		nil,
		client,
		"",
		""}

	go initiate(&sunny, initiateData, term, pvStore, statsStore, terminateCh)

	return

}

func initiate(sunny *sunnyDataProvider,
	initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore,
	terminateCh chan int) {

	// First request will start a session on the server
	// And give us cookies and viewstate that we need when logging in
	err := sunny.preLogin()
	if err != nil {
		term()
		return
	}
	err = sunny.login(initiateData.UserName, initiateData.Password)
	if err != nil {
		term()
		return
	}

	_, err = sunny.plantName()
	if err != nil {
		term()
		return
	}

	err = sunny.setPlantNo(initiateData.PlantNo)
	if err != nil {
		term()
		return
	}

	sunny.Plantname, err = sunny.plantName()
	if err != nil {
		term()
		return
	}
	log.Infof("Plant %s is now online", sunny.Plantname)

	go dataproviders.RunUpdates(
		&initiateData,
		func(id *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			pac, err := updatePacData(sunny.client)
			if err != nil {
				return err
			}
			pv.PowerAc = pac

			pv.LatestUpdate = nil
			return nil
		},
		func(id *dataproviders.InitiateData, pv *dataproviders.PvData) error {
			pvdaily, err := updateDailyProduction(sunny.client)
			if err != nil {
				return err
			}
			today, ok := pvdaily[nowDate()]
			if ok {
				pv.EnergyToday = today
			} else {
				pv.EnergyToday = 0
			}

			etotal, err := updateTotalProduction(sunny.client)
			if err != nil {
				return err
			}
			pv.EnergyTotal = etotal

			return nil
		},
		time.Second*5,
		time.Minute*5,
		time.Minute*30,
		terminateCh,
		term,
		MAX_ERRORS,
		statsStore,
		pvStore)

	return
}

func (c *sunnyDataProvider) preLogin() error {
	resp, err := c.client.Get(startUrl)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	defer resp.Body.Close()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in initiate", r)
		}
	}()

	log.Tracef("Got a %s reply on initiate cookies and viewstate", resp.Status)

	reg, err := regexp.Compile("<input type=\"hidden\" name=\"__VIEWSTATE\" id=\"__VIEWSTATE\" value=\"[^\"]*")
	if err != nil {
		log.Fail(err.Error())
	}
	b, _ := ioutil.ReadAll(resp.Body)
	found := reg.Find(b)
	c.viewstate = string(found[64:])

	log.Tracef("Viewstate in form: %s", c.viewstate)
	c.printCookies()
	return nil
}

func (c *sunnyDataProvider) login(username string, password string) error {
	formData := url.Values{}
	formData.Add("__VIEWSTATE", c.viewstate)
	formData.Add("ctl00$ContentPlaceHolder1$Logincontrol1$txtUserName", username)
	formData.Add("ctl00$ContentPlaceHolder1$Logincontrol1$txtPassword", password)
	formData.Add("ctl00$ContentPlaceHolder1$Logincontrol1$LoginBtn", "Login")
	log.Debugf("Posting to %s, with body: %s", loginUrl, formData.Encode())
	resp, err := c.client.PostForm(loginUrl, formData)
	if resp != nil && resp.StatusCode == 302 {
		log.Debug("Login success!")
	} else {
		if err != nil {
			log.Fail(err.Error())
			return err
		}
		defer resp.Body.Close()

		b, _ := ioutil.ReadAll(resp.Body)
		log.Failf("Login failed, http status codes was %s\n%s", resp.Status, b)
		return fmt.Errorf("Login to portal failed. Wrong username and password")
	}

	c.printCookies()

	return nil
}

func (c *sunnyDataProvider) setPlantNo(plantno string) error {
	// Try 5 times
	err := errors.New("")
	for i := 0; i < 5; i++ {
		err = c.sendPlantNoSwitchCommand(plantno)
		if err == nil {
			log.Debug("setPlantNo successfull!")
			return nil
		}
		log.Debugf("setPlantNo, try number %d", i)
	}
	return err
}

func (c *sunnyDataProvider) sendPlantNoSwitchCommand(plantno string) error {
	formData := url.Values{}
	formData.Add("__EVENTTARGET", fmt.Sprintf("ctl00$NavigationLeftMenuControl$0_%s", plantno))
	formData.Add("__VIEWSTATE", "")
	formData.Add("__EVENTARGUMENT", "")
	formData.Add("ctl00$_scrollPosHidden", "")
	formData.Add("LeftMenuNode_0", "1")
	formData.Add("LeftMenuNode_1", "1")
	formData.Add("LeftMenuNode_2", "0")
	//--
	//formData.Add("ctl00$HiddenPlantOID", "facb16e7-c40d-4316-a853-48c7620d1745")
	formData.Add("TabSwitchDeviceSelectionHid", "1")
	formData.Add("ctl00$ContentPlaceHolder1$FixPageWidth", "720")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$_datePicker$textBox", "6/7/2013")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$NavigateDivHidden", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$PlantName", "Guldnældevænget 33")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$SelectedIntervalID", "3")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$currentplantPowerWidgetContent$__dataFromCsharpPlantPowerWidgetControl",
		";currentPlantPowerPointerAngle:78;x_hours_ago:{0} hours ago;just_now:just now;one_hours_ago:an hour ago;some_seconds_ago:a few seconds ago;x_minutes_ago:{0} minutes ago")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$currentplantPowerWidgetContent$__dataToCsharpPlantPowerWidgetControl", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$googleMapsWidgetLoader$SubControl$googleMapsWidgetContent$googleMapAdr", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$googleMapsWidgetLoader$SubControl$googleMapsWidgetContent$googleMapLat", "54.765422")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$googleMapsWidgetLoader$SubControl$googleMapsWidgetContent$googleMapLng", "11.87549")

	c.printCookies()

	log.Debugf("Posting to %s, with body: %s", plantSelectUrl, formData)
	resp, err := c.client.PostForm(plantSelectUrl, formData)
	if resp != nil && resp.StatusCode == 302 {
		log.Debug("Plant selection success!")
		l, _ := resp.Location()
		log.Tracef("Received location as %s", *l)
	} else {
		if err != nil {
			log.Fail(err.Error())
			return err
		}
		defer resp.Body.Close()
		//b, _ :=
		ioutil.ReadAll(resp.Body)
		log.Failf("Plant selection failed, http status codes was %s\n", resp.Status)
		return fmt.Errorf("Switch to plantno %s failed.", plantno)
	}

	return nil
}

func updateTotalEnergyData(client *http.Client) (energyTotal float32, err error) {
	formData := url.Values{}
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$DashboardScriptManager",
		"ctl00$ContentPlaceHolder1$UserControlShowDashboard1$energyYieldWidgetLoader$AsyncLoadControlUpdatePanel|ctl00_ContentPlaceHolder1_UserControlShowDashboard1_energyYieldWidgetLoader_AsyncLoadControlUpdatePanel")
	formData.Add("__EVENTTARGET", "ctl00_ContentPlaceHolder1_UserControlShowDashboard1_energyYieldWidgetLoader_AsyncLoadControlUpdatePanel")
	formData.Add("__VIEWSTATE", "")
	formData.Add("__EVENTARGUMENT", "AsyncLoad_energyYieldWidgetLoader")
	formData.Add("ctl00$_scrollPosHidden", "")
	formData.Add("LeftMenuNode_0", "0")
	formData.Add("LeftMenuNode_1", "1")
	formData.Add("LeftMenuNode_2", "0")
	//--
	//formData.Add("ctl00$HiddenPlantOID", "facb16e7-c40d-4316-a853-48c7620d1745")
	formData.Add("TabSwitchDeviceSelectionHid", "1")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$currentplantPowerWidgetContent$__dataToCsharpPlantPowerWidgetControl", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$_datePicker$textBox", "6/7/2013")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$NavigateDivHidden", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$PlantName", "Guldnældevænget 33")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$SelectedIntervalID", "3")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$currentplantPowerWidgetContent$__dataFromCsharpPlantPowerWidgetControl",
		";currentPlantPowerPointerAngle:78;x_hours_ago:{0} hours ago;just_now:just now;one_hours_ago:an hour ago;some_seconds_ago:a few seconds ago;x_minutes_ago:{0} minutes ago")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$currentplantPowerWidgetContent$__dataToCsharpPlantPowerWidgetControl", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowDashboard1$UserControlShowEnergyAndPower1$UseIntervalHour", "0")

	log.Debugf("Posting to %s, with body: %s", plantSelectUrl, formData)
	resp, err := client.PostForm(plantSelectUrl, formData)

	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	log.Debugf("Response status for getting energy total was %s\n", resp.Status)

	log.Debugf("Response was %s", b)

	return

}

func (c *sunnyDataProvider) plantName() (name string, err error) {
	log.Debugf("Getting from %s", profileUrl)
	resp, err := c.client.Get(profileUrl)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in plantName", r)
		}
	}()
	b, _ := ioutil.ReadAll(resp.Body)

	log.Debugf("Received status %d on request", resp.StatusCode)
	//log.Tracef("%s", b)

	reg, err := regexp.Compile("<span id=\"ctl00_ContentPlaceHolder1_PlantProfileLabel\">.*</span>")
	if err != nil {
		log.Fail(err.Error())
		return
	}

	found := reg.Find(b)
	name = string(found[71 : len(found)-7])
	log.Debugf("Plantname was found as %s", name)
	return
}

func updatePacData(c *http.Client) (pac uint16, err error) {
	resp, err := c.Get(pacUrl)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	log.Debugf("Got a %s reply", resp.Status)
	jsonbytes, _ := ioutil.ReadAll(resp.Body)
	statusCode := resp.StatusCode
	if statusCode != 200 {
		err = fmt.Errorf(
			"Cannot load realtime values in dataprovider. Received http status %d", statusCode)
		return
	}
	log.Tracef("Received Pac json from sma: %s", jsonbytes)
	pacReply := smaPacReply{}
	err = json.Unmarshal(jsonbytes, &pacReply)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	if pacReply.Info.CurrentPlantTime != "" &&
		pacReply.Info.CurrentPlantPower == "" {
		// There was reply on the date but CurrentPlantPower
		// is not within the json string. This means the inverter is
		// in standby.. So we say thats ok, an sets the
		// current power production to 0
		pac = 0
	} else {
		pacint := 0
		pacint, err = strconv.Atoi(pacReply.Info.CurrentPlantPower)

		if err != nil {
			log.Fail(err.Error())
			return
		}
		pac = uint16(pacint)
	}

	return
}

func getCsvFile(client *http.Client, intervalId string) (csv []byte, err error) {
	log.Debugf("Getting from %s", csvPostUrl)
	resp, err := client.Get(csvPostUrl)

	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in updateDailyProduction", r)
		}
	}()
	b, _ := ioutil.ReadAll(resp.Body)

	log.Debugf("Received status %d on pre request", resp.StatusCode)

	reg, err := regexp.Compile("<input type=\"hidden\" name=\"__ctl00\\$ContentPlaceHolder1\\$UserControlShowEnergyAndPower1\\$_diagram_VIEWSTATE\" id=\"__ctl00\\$ContentPlaceHolder1\\$UserControlShowEnergyAndPower1\\$_diagram_VIEWSTATE\" value=\"[^\"]*")
	if err != nil {
		log.Fail(err.Error())
		return
	}

	found := reg.Find(b)

	viewstate := string(found[196:])
	log.Debugf("Viewstate was found as %s", viewstate)

	formData := url.Values{}
	//formData.Add("__EVENTTARGET", "")
	formData.Add("__EVENTARGUMENT", "")
	formData.Add("__ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$_diagram_VIEWSTATE", viewstate)
	formData.Add("__VIEWSTATE", "")
	formData.Add("__ASYNCPOST", "true")
	formData.Add("ctl00$_scrollPosHidden", "0")
	formData.Add("LeftMenuNode_0", "1")
	formData.Add("LeftMenuNode_1", "0")
	formData.Add("__EVENTTARGET", "ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$LinkButton_TabBack1")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$SelectedIntervalID", intervalId)
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$UseIntervalHour", "0")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$ImageButtonDownload.x", "10")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$ImageButtonDownload.y", "14")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$NavigateDivHidden", "")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$_datePicker$textBox",
		time.Now().Format(smaWebDateFormat))
	formData.Add("ctl00$ContentPlaceHolder1$FixPageWidth", "720")
	//formData.Add("ctl00$HiddenPlantOID", "facb16e7-c40d-4316-a853-48c7620d1745")
	log.Debugf("Posting to %s, with body: %s", csvPostUrl, formData)
	resp, err = client.PostForm(csvPostUrl, formData)
	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		log.Debugf("Post to %s success!", csvPostUrl)
	} else {
		if err != nil {
			err = fmt.Errorf("Post to %s failed, http status codes was %s\n%s", csvPostUrl, resp.Status, b)
		}
		return
	}
	defer resp.Body.Close()
	//c.printCookies()
	b, _ = ioutil.ReadAll(resp.Body)

	log.Debugf("Getting from %s", csvUrl)
	resp, err = client.Get(csvUrl)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	//csv, _ := ioutil.ReadAll(resp.Body)

	log.Debugf("Received status %d on csv request", resp.StatusCode)

	csv, _ = ioutil.ReadAll(resp.Body)
	log.Tracef("CSV received: %s", csv)

	return

}

func updateDailyProduction(client *http.Client) (pvDaily dataproviders.PvDataDaily, err error) {
	// Jump over first line
	csv, err := getCsvFile(client, "4")
	if err != nil {
		return
	}

	reader := bufio.NewReader(bytes.NewReader(csv))
	_, _ = reader.ReadString('\n')
	pvDaily = make(map[string]uint16)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		cols := strings.Split(line, ";")
		if len(cols) < 2 {
			continue
		}
		log.Tracef("Date %s, production %s", cols[0], cols[1])
		prod, _ := strconv.ParseFloat(cols[1], 64)
		pvDaily[parseSmaDateToKey(cols[0])] = uint16(prod * 1000)
	}
	return
}

func updateTotalProduction(client *http.Client) (value float32, err error) {
	// Jump over first line
	csv, err := getCsvFile(client, "6")
	if err != nil {
		log.Failf("Error in updateTotalProduction: %s", err.Error())
		return
	}
	reader := bufio.NewReader(bytes.NewReader(csv))

	_, _ = reader.ReadString('\n')

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		log.Tracef("%s", line)
		cols := strings.Split(line, ";")
		log.Tracef("Cols: %s", cols)
		if len(cols) < 3 {
			continue
		}
		log.Tracef("Year %s, production %s", cols[0], cols[1])
		value64, _ := strconv.ParseFloat(cols[1], 64)
		value += float32(value64)

	}
	return
}

func parseSmaDateToKey(date string) string {
	t, err := time.Parse(smaCsvDateFormat, date)
	if err != nil {
		log.Info(err.Error())
	}
	return t.Format(keyDateFormat)
}

func nowDate() string {
	return time.Now().Format(keyDateFormat)
}

func (c *sunnyDataProvider) printCookies() {
	log.Trace("Cookies in store is now:")

	for _, cookie := range c.client.Jar.Cookies(nil) {
		log.Tracef("    %s: %s", cookie.Name, cookie.Value)
	}

}
