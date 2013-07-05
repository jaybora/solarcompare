// +build appengine
package serverpart

import (
	"logger"
	"net/http"
	"serverpart/handlers"
)

var log = logger.NewLogger(logger.INFO, "app.go ")

func init() {
	http.HandleFunc("/api/v1/plant", handlers.PlantHandler)
	http.HandleFunc("/api/v1/plant/", handlers.PlantHandler)
	http.HandleFunc("/api/v1/dpunit", handlers.DpUnitHandler)
	http.HandleFunc("/api/v1/dpunit/", handlers.DpUnitHandler)
	http.HandleFunc("/api/v1/auth/login", handlers.AuthRedirectLoginHandler)
	http.HandleFunc("/api/v1/auth/logout", handlers.AuthRedirectLogoutHandler)
	http.HandleFunc("/api/v1/auth/user", handlers.AuthUserHandler)
	http.HandleFunc("/api/v1/dataprovider", handlers.DataProviderHandler)
	http.HandleFunc("/api/v1/dataprovider/", handlers.DataProviderHandler)

	//	http.HandleFunc("/plant/pvdata", pvdatahandler)
	//	http.HandleFunc("/dpunit", dpunithandler)

}
