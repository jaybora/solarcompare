// +build appengine
package handlers

import (
	"appengine"
	"dataproviders"
	"encoding/json"
	"net/http"
)

//List known dataproviders
func DataProviderHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	json, err := json.MarshalIndent(dataproviders.DataProviders, "", "  ")
	if err != nil {
		c.Errorf("Error in marshalling known dataproviders, err %s", err.Error())
	}
	w.Header().Add("Content-type", "application/json")
	w.Write(json)

}
