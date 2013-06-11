// +build appengine
package handlers

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"encoding/json"
	"fmt"
	"goon"
	"io/ioutil"
	"net/http"
	"time"
)

type DataProviderUnit struct {
	ConnectURL   string `goon:"id"`
	RegisterTime time.Time
}

/*
 Get a DpUnit available for next dataprovider setup
*/
func DpUnitAvailable(r *http.Request) *DataProviderUnit {
	c := appengine.NewContext(r)

	i, err := memcache.Get(c, "DpuAvail")
	if err == nil {
		dpu := DataProviderUnit{}
		json.Unmarshal(i.Value, &dpu)
		c.Debugf("Got dpu avail from memcache..")
		return &dpu
	}

	g := goon.NewGoon(r)
	q := datastore.NewQuery("DataProviderUnit").
		Limit(1)

	iterator := g.Run(q)
	for {
		dpu := DataProviderUnit{}
		_, err := iterator.Next(&dpu)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf(err.Error())
			return nil
		}

		b, _ := json.Marshal(&dpu)
		item := &memcache.Item{
			Key:   "DpuAvail",
			Value: b}

		c.Debugf("Saving dpu avail to memcache..")
		err = memcache.Set(c, item)
		if err != nil {
			c.Errorf(err.Error())
			return nil
		}

		return &dpu

	}

	return nil
}

func DpUnitHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		handleGetDpu(w, r)
	case "POST":
		handlePutDpu(w, r)
	case "PUT":
		handlePutDpu(w, r)
	case "DELETE":
		handleDeleteDpu(w, r)
	}
}

func handleGetDpu(w http.ResponseWriter, r *http.Request) {
	g := goon.NewGoon(r)
	q := datastore.NewQuery("DataProviderUnit").
		Limit(10)

	w.Write([]byte("<html><h1>Registered dataproviderunits:</h1>"))
	w.Write([]byte("<table border=1><th>URL<th>Register timer</th>"))

	iterator := g.Run(q)
	for {
		dpu := DataProviderUnit{}
		_, err := iterator.Next(&dpu)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("%s", err.Error()),
				http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "<tr><td>%s<td>%s</tr>", dpu.ConnectURL, dpu.RegisterTime)

	}

	w.Write([]byte("</table></html>"))

}

func handlePutDpu(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	jsonbytes, _ := ioutil.ReadAll(r.Body)
	dpu := DataProviderUnit{}
	err := json.Unmarshal(jsonbytes, &dpu)
	if err != nil {
		http.Error(w, fmt.Sprintf("DataProviderUnit data could not be unmarshalled"),
			http.StatusBadRequest)
		return
	}

	dpu.RegisterTime = time.Now()

	if err := g.Put(&dpu); err != nil {
		c.Errorf("Could not write dpu %s: %s, %s", dpu, err.Error())
		http.Error(w, fmt.Sprintf("DataProviderUnit could be stored"),
			http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Ok"))
}
func handleDeleteDpu(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.NewGoon(r)
	jsonbytes, _ := ioutil.ReadAll(r.Body)
	dpu := DataProviderUnit{}
	err := json.Unmarshal(jsonbytes, &dpu)
	if err != nil {
		http.Error(w, fmt.Sprintf("DataProviderUnit data could not be unmarshalled"),
			http.StatusBadRequest)
		return
	}

	if err = g.Delete(g.Key(&dpu)); err != nil {
		c.Errorf("Could not delete dpu %s: %s", dpu, err.Error())
		http.Error(w, fmt.Sprintf("DataProviderUnit could not be deleted"),
			http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Ok"))
}
