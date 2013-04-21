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
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore) (err error) {
	switch implType {
	case dataproviders.FJY:
		JFY.NewDataProvider(init, term, newClient(), pvStore, statsStore)
		//provider = &jfy
		return
	case dataproviders.SunnyPortal:
		_, err2 := sunnyportal.NewDataProvider(init, term, newClient(), 
		                                           pvStore, statsStore)
		if err2 != nil {
			err = err2
			return
		}
		//provider = &sunny
		return
	case dataproviders.Suntrol:
		suntrol.NewDataProvider(init, term, newClient(), pvStore, statsStore)
		//provider = &dp
		return
	case dataproviders.Danfoss:
		danfoss.NewDataProvider(init, term, newClient(), pvStore, statsStore)
		//provider = &dp
		return
	err = fmt.Errorf("No provider found for %d", implType)
	}
	return
}
