package web

import (
	"net/http"
	"fmt"
)


func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Only <a href='/json'>json</a> output is available at the moment!")
}


