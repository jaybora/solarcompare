package web

import (
	"net/http"
	"text/template"
)


func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	
	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Cannot find html template, " + err.Error(), http.StatusInternalServerError)
		return 
	}
	
	t.Execute(w, nil)
}