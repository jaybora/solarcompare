// +build !appengine

package main

import (
	"bytes"
	"controller"
	"dataproviders"
	"encoding/json"
	"flag"
	"fmt"
	"httpclient"
	"io/ioutil"
	"logger"
	"net/http"
	"stores"
	"time"
	"web"
)
import _ "net/http/pprof"

var (
	publicUrl = flag.String("publicurl", "", "The URL at wich the server can connect to this dataprovider unit")
	httpPort  = flag.String("port", "8080", "The port of where the local webserver should run")
	serverUrl = flag.String("serverurl", "http://solar-compare.appspot.com", "The URL of the server")
)
var log = logger.NewLogger(logger.DEBUG, "main: ")

type DataProviderUnit struct {
	ConnectURL string
}

// DataProviderUnit
// Must implement:
// - a IAmReadyHere command transmitted to server at reguler intervals
// - a webserver that can:
// - - Receive an order from server to begin serving data from a given plant
// - - Receive an order to display active providers

// Default Request Handler
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>")
	fmt.Fprintf(w, "This is dataprovider unit running at port %s<br/>", *httpPort)
	fmt.Fprintf(w, "Public URL is set to %s<br/>", *publicUrl)
	fmt.Fprintf(w, "<a href='plant'>Show active plants</a>")
	fmt.Fprintf(w, "</body></html>")
}

func main() {

	flag.Parse()
	if *publicUrl == "" {
		fmt.Println("Need to specify parameters...")
		flag.PrintDefaults()
		return
	}
	fmt.Printf("Using %s as public URL\n", *publicUrl)

	startWebServer()
}

func uploadPvData(plantkey *string, pv *dataproviders.PvData) {
	b := pv.ToJson()
	url := fmt.Sprintf("%s/plant/%s/pvdata", *serverUrl, *plantkey)
	log.Debugf("Posting %s to %s", b, url)
	resp, err := httpclient.NewClient().Post(url, "", bytes.NewBuffer(b))
	if err != nil {
		log.Failf("Could not send to server due to %s", err.Error())

	} else {
		log.Debugf("Received %s on that", resp.Status)
		defer resp.Body.Close()
	}

}

func startWebServer() {
	pvStore := stores.NewPvStore(uploadPvData)
	plants := stores.NewPlantStore()
	controller := controller.NewController(httpclient.NewClient,
		pvStore,
		stores.StatsStoreFile{},
		func(plantKey *string) {
			plants.Remove(*plantKey)
		})
	log.Debugf("New controller is %s", controller)

	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/plant", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants, pvStore, true)
	})
	http.HandleFunc("/plant/", func(w http.ResponseWriter, r *http.Request) {
		web.PlantHandler(w, r, &controller, plants, pvStore, true)
	})
	go uploadPublicUrl()

	fmt.Printf("Webserver running on port %s\n", *httpPort)
	srv := http.Server{Addr: fmt.Sprintf(":%s", *httpPort), ReadTimeout: 10 * time.Second}
	err := srv.ListenAndServe()

	if err != nil {
		log.Fail(err.Error())
	}
}

func uploadPublicUrl() {
	url := fmt.Sprintf("%s/dpunit", *serverUrl)
	for {

		log.Infof("Uploading our public url %s to server %s", *publicUrl, url)
		dpu := DataProviderUnit{*publicUrl}
		b, err := json.Marshal(&dpu)
		if err != nil {
			log.Failf("UploadPoublic error %s", err.Error())
		}
		log.Debugf("Posting %s to %s", b, url)
		resp, err := httpclient.NewClient().Post(url, "", bytes.NewBuffer(b))
		if err != nil {
			log.Failf("Could not send to server due to %s", err.Error())

		} else {
			log.Debugf("Received %s on that", resp.Status)
			ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
		}

		c := time.Tick(5 * time.Minute)
		select {
		case <-c:
		}
	}
}
