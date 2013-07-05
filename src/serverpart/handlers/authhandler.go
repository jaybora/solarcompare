// +build appengine
package handlers

import (
	"appengine"
	"appengine/user"
	"encoding/json"
	"fmt"
	"net/http"
)

// This makes a redirect to a login page
func AuthRedirectLoginHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	url, err := user.LoginURL(c, "/")
	if err != nil {
		c.Errorf("Could not get loginurl due to %s", err.Error())
		http.Error(w, fmt.Sprintf(err.Error()),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusMovedPermanently)

}

func AuthRedirectLogoutHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	url, err := user.LogoutURL(c, "/")
	if err != nil {
		c.Errorf("Could not get logouturl due to %s", err.Error())
		http.Error(w, fmt.Sprintf(err.Error()),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusMovedPermanently)

}

// Return userinfo as a json
func AuthUserHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.Write(b)
}
