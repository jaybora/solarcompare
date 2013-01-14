package sunnyportal

import (
	"crypto/tls"
	"dataproviders"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type sunnyDataProvider struct {
	InitiateData   dataproviders.InitiateData
	latestReqCh    chan chan dataproviders.PvData
	latestUpdateCh chan dataproviders.PvData
	terminateCh    chan int
	latestErr      error
	client         *http.Client
	viewstate      string //Something that sma portal uses, must be posted to login

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

var log = logger.NewLogger(logger.TRACE, "Dataprovider: SunnyPortal:")

const MAX_ERRORS = 10
const INACTIVE_TIMOUT = 300 //secs

const startUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const loginUrl = "http://www.sunnyportal.com/Templates/Start.aspx"
const pacUrl = "http://www.sunnyportal.com/Dashboard"
const csvPostUrl = "http://sunnyportal.com/FixedPages/EnergyAndPower.aspx"
const csvUrl = "http://sunnyportal.com/Templates/DownloadDiagram.aspx?down=diag"

const keyDateFormat = "20060102"
const smaCsvDateFormat = "1/2/06"
const smaWebDateFormat = "01/02/2006"

func (sunny *sunnyDataProvider) Name() string {
	return "SunnyPortal"
}

func NewDataProvider(initiateData dataproviders.InitiateData, term dataproviders.TerminateCallback) sunnyDataProvider {
	log.Debug("New dataprovider")
	jar := new(Jar)
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Jar: jar}

	sunny := sunnyDataProvider{initiateData,
		make(chan chan dataproviders.PvData),
		make(chan dataproviders.PvData),
		make(chan int),
		nil,
		client,
		""}

	err := sunny.initiate()

	go dataproviders.RunUpdates(sunny.client, sunny.latestUpdateCh, term, sunny.terminateCh)
	go dataproviders.LatestPvData(sunny.latestReqCh, sunny.latestUpdateCh, sunny.terminateCh)

	return sunny
}

func (c *sunnyDataProvider) initiate() error {
	resp, err := c.client.Get(startUrl)
	if err != nil {
		log.Fail(err.Error())
		return err
	}
	defer resp.Body.Close()
	log.Tracef("Got a %s reply", resp.Status)

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

func (c *sunnyDataProvider) pac() (json []byte, err error) {
	resp, err := c.client.Get(pacUrl)
	if err != nil {
		log.Fail(err.Error())
		return
	}
	defer resp.Body.Close()
	log.Debugf("Got a %s reply", resp.Status)
	json, _ = ioutil.ReadAll(resp.Body)
	statusCode := resp.StatusCode
	if statusCode != 200 {
		err := fmt.Errorf(
			"Cannot load realtime values in dataprovider. Received http status %s", statusCode)
	}
	return
}

func (c *sunnyDataProvider) DailyProduction() (pvDaily pv.PvDataDaily, err error) {
	log.Printf("Getting from %s", csvPostUrl)
	resp, err := c.client.Get(csvPostUrl)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	log.Printf("Received status %d on pre request", resp.StatusCode)

	reg, err := regexp.Compile("<input type=\"hidden\" name=\"__ctl00\\$ContentPlaceHolder1\\$UserControlShowEnergyAndPower1\\$_diagram_VIEWSTATE\" id=\"__ctl00\\$ContentPlaceHolder1\\$UserControlShowEnergyAndPower1\\$_diagram_VIEWSTATE\" value=\"[^\"]*")
	if err != nil {
		log.Fatal(err.Error())
	}

	found := reg.Find(b)
	viewstate := string(found[196:])
	log.Printf("Viewstate was found as %s", viewstate)

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
	formData.Add("ctl00$HiddenPlantOID", "facb16e7-c40d-4316-a853-48c7620d1745")
	log.Printf("Posting to %s, with body: %s", csvPostUrl, formData)
	resp, err = c.client.PostForm(csvPostUrl, formData)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()
	c.printCookies()
	b, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 || resp.StatusCode == 302 {
		log.Printf("Post to %s success!", csvPostUrl)
	} else {
		err = fmt.Errorf("Post to %s failed, http status codes was %s\n%s", csvPostUrl, resp.Status, b)
		return
	}

	log.Printf("Getting from %s", csvUrl)
	resp, err = c.client.Get(csvUrl)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer resp.Body.Close()
	//csv, _ := ioutil.ReadAll(resp.Body)

	log.Printf("Received status %s on csv request", resp.StatusCode)

	reader := bufio.NewReader(resp.Body)
	log.Print("CSV received:")
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
		log.Printf("Kl %s, production %s", cols[0], cols[1])
		prod, _ := strconv.ParseFloat(cols[1], 64)
		pvDaily[parseSmaDateToKey(cols[0])] = uint16(prod * 1000)
	}

	return
}

func parseSmaDateToKey(date string) string {
	t, err := time.Parse(smaCsvDateFormat, date)
	if err != nil {
		log.Fatal(err.Error())
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
	log.Tracef("Returning PvData as %s", pv)
	return
}

func (c *sunnyDataProvider) printCookies() {
	log.Print("Cookies in store is now:")
	for _, cookie := range c.client.Jar.Cookies(nil) {
		log.Printf("    %s: %s", cookie.Name, cookie.Value)
	}

}
