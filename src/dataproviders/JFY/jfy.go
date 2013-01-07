package JFY

import (
	"dataproviders"
	"log"
	"net/http"
	"encoding/json"
	"io/ioutil"
)

type jfyDataProvider struct {
	InitiateData dataproviders.InitiateData
	latest       dataproviders.PvData
	client       *http.Client
}

const GetUrl="http://cts.jbr.dk:81/json"

func (jfy jfyDataProvider) Name() string {
	return "JFY"
}

func NewDataProvider(initiateData dataproviders.InitiateData) jfyDataProvider {
	log.Printf("New JFY dataprovider")
	client := &http.Client{}

	return jfyDataProvider{initiateData, dataproviders.PvData{},client}
}

// Get PvData
// TODO: Cache the data
func (jfy jfyDataProvider) PvData() (pv dataproviders.PvData, err error) {
	resp, err := jfy.client.Get(GetUrl)
	if err != nil {
		log.Printf(err.Error())
		return 
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(b, &pv)
	jfy.latest = pv
	return
}
