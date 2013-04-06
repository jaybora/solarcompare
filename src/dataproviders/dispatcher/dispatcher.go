package dispatcher

import (
	"dataproviders"
	"dataproviders/JFY"
	"dataproviders/sunnyportal"
	"dataproviders/suntrol"
	"dataproviders/danfoss"
	"fmt"
	"net/http"
)

type NewClient func() *http.Client

func Provider(implType int,
	init dataproviders.InitiateData,
	term dataproviders.TerminateCallback,
	newClient NewClient,
	pvDataUpdatedEvent dataproviders.PvDataUpdatedEvent,
	statsStore dataproviders.PlantStatsStore) (
	provider dataproviders.DataProvider, err error) {
	switch implType {
	case dataproviders.FJY:
		jfy := JFY.NewDataProvider(init, term, newClient(), pvDataUpdatedEvent, statsStore)
		provider = &jfy
		return
	case dataproviders.SunnyPortal:
		sunny, err2 := sunnyportal.NewDataProvider(init, term, newClient(), 
		                                           pvDataUpdatedEvent, statsStore)
		if err2 != nil {
			err = err2
			return
		}
		provider = &sunny
		return
	case dataproviders.Suntrol:
		dp := suntrol.NewDataProvider(init, term, newClient(), pvDataUpdatedEvent, statsStore)
		provider = &dp
		return
	case dataproviders.Danfoss:
		dp := danfoss.NewDataProvider(init, term, newClient(), pvDataUpdatedEvent, statsStore)
		provider = &dp
		return
	err = fmt.Errorf("No provider found for %d", implType)
	}
	return
}
