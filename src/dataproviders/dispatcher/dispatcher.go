package dispatcher

import (
	"dataproviders"
	"dataproviders/JFY"
	"dataproviders/sunnyportal"
	"fmt"
)

func Provider(implType int,
	init dataproviders.InitiateData,
	term dataproviders.TerminateCallback) (
	provider dataproviders.DataProvider, err error) {
	switch implType {
	case dataproviders.FJY:
		jfy := JFY.NewDataProvider(init, term)
		provider = &jfy
		return
	case dataproviders.SunnyPortal:
		
		sunny, err2 := sunnyportal.NewDataProvider(init, term)
		if err2 != nil {
			err = err2
			return
		}
		provider = &sunny
		return
	}
	err = fmt.Errorf("No provider found for %d", implType)
	return
}
