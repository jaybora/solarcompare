package web

import (
	"fmt"
	"net/http"
)


// Default Request Handler
func PlantHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "<h1>Hello %s!</h1>", r.URL.Path[1:])
}

