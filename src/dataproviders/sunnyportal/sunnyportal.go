package sunnyportal

import (
	"bufio"
	"crypto/tls"
	"dataproviders"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"net"
)

type sunnyDataProvider struct {
	InitiateData   dataproviders.InitiateData
	latestReqCh    chan chan dataproviders.PvData
	latestUpdateCh chan dataproviders.PvData
	terminateCh    chan int
	latestErr      error
	client         *http.Client
	viewstate      string //Something that sma portal uses, must be posted to login
	plantname      string

}

type smaPacReply struct {
	CurrentPlantPower string `json:"currentPlantPower"`
}

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		jar.cookies = append(jar.cookies, cookie)
	}
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

var log = logger.NewLogger(logger.INFO, "Dataprovider: SunnyPortal:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 300 //secs

const startUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const loginUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const profileUrl = "http://www.sunnyportal.com/FixedPages/PlantProfile.aspx"
const pacUrl = "http://www.sunnyportal.com/Dashboard"
const csvPostUrl = "http://sunnyportal.com/FixedPages/EnergyAndPower.aspx"
const csvUrl = "http://sunnyportal.com/Templates/DownloadDiagram.aspx?down=diag"
const plantSelectUrl = "http://sunnyportal.com/FixedPages/PlantProfile.aspx"

const keyDateFormat = "20060102"
const smaCsvDateFormat = "1/2/06"
const smaWebDateFormat = "1/2/2006"

func (sunny *sunnyDataProvider) Name() string {
	return sunny.plantname
}

func NewDataProvider(initiateData dataproviders.InitiateData,
	term dataproviders.TerminateCallback) (sunny sunnyDataProvider, err error) {

	log.Debug("New dataprovider")
	jar := new(Jar)
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: func(netw, addr string) (net.Conn, error) {
			// Specify timeout/deadline
			deadline := time.Now().Add(8 * time.Second)
			c, err := net.DialTimeout(netw, addr, time.Second)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		}}
	client := &http.Client{Transport: transport, Jar: jar}

	sunny = sunnyDataProvider{initiateData,
		make(chan chan dataproviders.PvData),
		make(chan dataproviders.PvData),
		make(chan int),
		nil,
		client,
		"",
		""}

	// First request will start a session on the server
	// And give us cookies and viewstate that we need when logging in
	err = sunny.initiate()
	if err != nil {
		return
	}
	err = sunny.login(initiateData.UserName, initiateData.Password)
	if err != nil {
		return
	}

	_, err = sunny.plantName()
	if err != nil {
		return
	}

	err = sunny.setPlantNo(initiateData.PlantNo)
	if err != nil {
		return
	}
	
	sunny.plantname, err = sunny.plantName()
	if err != nil {
		return
	}
	log.Infof("Plant %s is now online", sunny.plantname)

	go dataproviders.RunUpdates(
		func(pvin dataproviders.PvData) (pv dataproviders.PvData, err error) {
			pac, err := updatePacData(client)
			pvin.PowerAc = pac
			pv = pvin
			return
		},
		func(pvin dataproviders.PvData) (pv dataproviders.PvData, err error) {
			pvdaily, err := updateDailyProduction(client)
			today, ok := pvdaily[nowDate()]
			if ok {
				pvin.EnergyToday = today
			} else {
				pvin.EnergyToday = 9999
			}
			pv = pvin
			return
		},
		time.Second*5,
		time.Minute*5,
		time.Minute*30,
		sunny.latestUpdateCh,
		sunny.latestReqCh,
		term,
		sunny.terminateCh,
		MAX_ERRORS,
		sunny.plantname)

	go dataproviders.LatestPvData(
		sunny.latestReqCh,
		sunny.latestUpdateCh,
		sunny.terminateCh)

	return
}

func (c *sunnyDataProvider) initiate() error {
	resp, err := c.client.Get(startUrl)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	defer resp.Body.Close()
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
	log.Debugf("Posting to %s, with body: %s", loginUrl, formData)
	resp, err := c.client.PostForm(loginUrl, formData)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	
	c.printCookies()

	if resp.StatusCode == 302 {
		log.Debug("Login success!")
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Failf("Login failed, http status codes was %s\n%s", resp.Status, b)
		return fmt.Errorf("Login to portal failed. Wrong username and password")
	}
	return nil
}

func (c *sunnyDataProvider) setPlantNo(plantno string) error {
	formData := url.Values{}
	formData.Add("__EVENTTARGET", fmt.Sprintf("ctl00$NavigationLeftMenuControl$0_%s", plantno))
	//formData.Add("ctl00$HiddenPlantOID", "facb16e7-c40d-4316-a853-48c7620d1745")
	formData.Add("__VIEWSTATE", "")
	formData.Add("__EVENTARGUMENT", "")
	formData.Add("ctl00$_scrollPosHidden", "")
	formData.Add("LeftMenuNode_0", "1")
	formData.Add("LeftMenuNode_1", "1")
	formData.Add("LeftMenuNode_2", "0")

	log.Debugf("Posting to %s, with body: %s", plantSelectUrl, formData)
	resp, err := c.client.PostForm(plantSelectUrl, formData)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 302 {
		log.Debug("Plant selection success!")
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Failf("Plant selection failed, http status codes was %s\n%s", resp.Status, b)
		return fmt.Errorf("Switch to plantno %s failed.", plantno)
	}
	return nil
}


func  (c *sunnyDataProvider) plantName() (name string, err error) {
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
	name = string(found[71:len(found)-7])
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
	pacint, err := strconv.Atoi(pacReply.CurrentPlantPower)

	if err != nil {
		log.Fail(err.Error())
		return
	}
	pac = uint16(pacint)
	return
}

func updateDailyProduction(client *http.Client) (pvDaily dataproviders.PvDataDaily, err error) {
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
	formData.Add("ctl00$_scrollPosHidden", "0")
	formData.Add("LeftMenuNode_0", "1")
	formData.Add("LeftMenuNode_1", "0")
	formData.Add("__EVENTTARGET", "ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$LinkButton_TabBack1")
	formData.Add("ctl00$ContentPlaceHolder1$UserControlShowEnergyAndPower1$SelectedIntervalID", "4")
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
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	//c.printCookies()
	b, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		log.Debugf("Post to %s success!", csvPostUrl)
	} else {
		err = fmt.Errorf("Post to %s failed, http status codes was %s\n%s", csvPostUrl, resp.Status, b)
		return
	}

	log.Debugf("Getting from %s", csvUrl)
	resp, err = client.Get(csvUrl)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	//csv, _ := ioutil.ReadAll(resp.Body)

	log.Debugf("Received status %s on csv request", resp.StatusCode)

	reader := bufio.NewReader(resp.Body)
	log.Debug("CSV received:")
	// Jump over first line
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

// Get latest PvData
func (sunny *sunnyDataProvider) PvData() (pv dataproviders.PvData, err error) {
	reqCh := make(chan dataproviders.PvData)
	sunny.latestReqCh <- reqCh
	pv = <-reqCh
	log.Tracef("Returning PvData as %s", pv.ToJson())
	return
}

func (c *sunnyDataProvider) printCookies() {
	log.Trace("Cookies in store is now:")
	for _, cookie := range c.client.Jar.Cookies(nil) {
		log.Tracef("    %s: %s", cookie.Name, cookie.Value)
	}

}
