package dataproviders

import (
	"../web"
)

type InitiateData struct {
	UserName string
	Password string
}

type DataProvider interface {
	Name() string
	Initiate(initiateData InitiateData)
	PvData() web.PvData
}
