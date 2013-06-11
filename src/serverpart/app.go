// +build appengine
package serverpart

import (
	"logger"
	"net/http"
	"serverpart/handlers"
)

var log = logger.NewLogger(logger.INFO, "app.go ")

func init() {
	http.HandleFunc("/plant", handlers.PlantHandler)
	http.HandleFunc("/plant/", handlers.PlantHandler)
	http.HandleFunc("/dpunit", handlers.DpUnitHandler)
	http.HandleFunc("/dpunit/", handlers.DpUnitHandler)
	http.HandleFunc("/auth/login", handlers.AuthRedirectLoginHandler)
	http.HandleFunc("/auth/logout", handlers.AuthRedirectLogoutHandler)
	http.HandleFunc("/auth/user", handlers.AuthUserHandler)

	//	http.HandleFunc("/plant/pvdata", pvdatahandler)
	//	http.HandleFunc("/dpunit", dpunithandler)

}
