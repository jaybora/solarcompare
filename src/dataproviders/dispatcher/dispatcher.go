package dispatcher

import (
	"dataproviders"
	"dataproviders/JFY"
)

func Provider(implType int, init dataproviders.InitiateData) dataproviders.DataProvider {
	switch implType {
		case dataproviders.FJY :
			return JFY.NewDataProvider(init)
	}
	return nil
}
