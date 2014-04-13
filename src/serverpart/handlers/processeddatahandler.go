// +build appengine

package handlers

import (
	"appengine"
	"fmt"
	"net/http"
	"web"
)

// This handler is called by app engine cron to regular
// log data to datastore

func ProcessedDataHandler(w http.ResponseWriter, r *http.Request) {
	// This handler looks at the urlparam "processtype" for
	// what action to take:
	// daily to sum up

	c := appengine.NewContext(r)
	if r.URL.Query().Get("processtype") == "daily" {
		// TODO: Make some processing

	} else {
		keypos := 2
		if !appengine.IsDevAppServer() {
			keypos += 2
		}

		plantkey := web.PlantKey(r.URL.String(), keypos)
		if plantkey != "" {
			p, err := Plant(r, &plantkey)

			if err != nil {
				c.Infof("Could not find plat due to %s", err.Error())
				http.Error(w, fmt.Sprintf(err.Error()),
					http.StatusInternalServerError)
				return
			}

			key := r.URL.Query().Get("key")

			data, _ := getProcessedDataEntity(r, p, &key)
			w.Header().Add("Content-type", "application/json")
			w.Write(data.Json)
			return

		} else {

			http.Error(w, "Currently url parameter can only be ?processtype=daily",
				http.StatusBadRequest)
		}
	}

}
