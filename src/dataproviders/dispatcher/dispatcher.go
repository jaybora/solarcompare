package dispatcher

import (
	"dataproviders"
	"dataproviders/JFY"
	"fmt"
)


func Provider(implType int, init dataproviders.InitiateData, term dataproviders.TerminateCallback) (provider dataproviders.DataProvider, err error) {
	switch implType {
		case dataproviders.FJY :
			jfy := JFY.NewDataProvider(init, term)
			provider = &jfy
			return
		case dataproviders.SunnyPortal:
			sunny := sunnyportal.NewDataProvider(init, term)
			provider = &sunny
			return
	}
	err = fmt.Errorf("No provider found for %d", implType)
	return
}
