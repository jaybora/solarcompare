package dataproviders

import (
)


type InitiateData struct {
	UserName string
	Password string
}


type DataProvider interface {
	Name() string
	PvData() (pv PvData, err error)
}

type TerminateCallback func()

// List of all known dataproviders
// This would be nice if we could autodetect them somehow
const (
	FJY = iota
	SunnyPortal = iota
)
