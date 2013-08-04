package dispatcher

import (
	"dataproviders"
	"dataproviders/JFY"
	"dataproviders/danfoss"
	"dataproviders/kostal"
	"dataproviders/solaredge"
	"dataproviders/sunnyportal"
	"dataproviders/suntrol"
	"fmt"
	"net/http"
)

type NewClient func() *http.Client

func Provider(implType int,
	init dataproviders.InitiateData,
	term dataproviders.TerminateCallback,
	terminateCh chan int,
	newClient NewClient,
	pvStore dataproviders.PvStore,
	statsStore dataproviders.PlantStatsStore) (err error) {
	switch implType {
	case dataproviders.FJY:
		JFY.NewDataProvider(init, term, newClient(), pvStore, statsStore, terminateCh)
		//provider = &jfy
		return
	case dataproviders.SunnyPortal:
		_, err2 := sunnyportal.NewDataProvider(init, term, newClient(),
			pvStore, statsStore, terminateCh)
		if err2 != nil {
			err = err2
			return
		}
		//provider = &sunny
		return
	case dataproviders.Suntrol:
		suntrol.NewDataProvider(init, term, newClient(), pvStore, statsStore, terminateCh)
		//provider = &dp
		return
	case dataproviders.Danfoss:
		danfoss.NewDataProvider(init, term, newClient(), pvStore, statsStore, terminateCh)
		//provider = &dp
		return
	case dataproviders.Kostal:
		kostal.NewDataProvider(init, term, newClient(), pvStore, statsStore, terminateCh)
		//provider = &dp
		return
	case dataproviders.Solaredge:
		solaredge.NewDataProvider(init, term, newClient(), pvStore, statsStore, terminateCh)
		//provider = &dp
		return
	default:
		err = fmt.Errorf("No provider found for %d", implType)
	}
	return
}
