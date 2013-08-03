package solaredge

import (
	"dataproviders"
	"httpclient"
	"testing"
)

/*
To run test export GOPATH=/Users/jbr/github/local/solarcompare
then
go test -test.v dataproviders/solaredge
*/

type PlantStatsStore struct {
}

func (p PlantStatsStore) LoadStats(plantkey string) dataproviders.PlantStats {
	return dataproviders.PlantStats{}
}
func (p PlantStatsStore) SaveStats(plantkey string, pv *dataproviders.PvData) {}

func Test_Connection(t *testing.T) {
	t.Log("Trying to launch provider...")
	pv := dataproviders.PvData{}
	err := updatePvData(httpclient.NewClient(),
		&dataproviders.InitiateData{"jan", "", "", "22175", ""},
		&pv)

	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Pv data is %s", pv.ToJson())

	/*

		if plantname == "" || plantname != "LilleStoreLarsen" {
			t.Errorf("Incorrect plant name recieved from sunnyportal = '%s'", plantname)
		}
		pac, err := updatePacData(p.client)
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("Pac data is %d", pac)
	*/
	return
}
