package danfoss

import (
	"testing"
	"httpclient"
	"dataproviders"
)

/*
To run test export GOPATH=/Users/jbr/github/local/solarcompare
then
go test -test.v dataproviders/danfoss
*/

type PlantStatsStore struct {
}

func (p PlantStatsStore) LoadStats(plantkey string) dataproviders.PlantStats {return dataproviders.PlantStats{}}
func (p PlantStatsStore) SaveStats(plantkey string, pv *dataproviders.PvData) {}


func Test_Connection (t *testing.T) {
	t.Log("Trying to launch provider...")
	pv, err := updatePvData(httpclient.NewClient(), &dataproviders.InitiateData{"jan", "anonym", "anonym", "", "5.103.131.3"})
	
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

