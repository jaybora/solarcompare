package main 

import (
    "net/http"
    "web"
)


func main() {
    http.HandleFunc("/plant", web.PlantHandler)
    http.ListenAndServe(":8080", nil)
}

