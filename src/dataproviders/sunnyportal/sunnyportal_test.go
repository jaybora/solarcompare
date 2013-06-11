package sunnyportal

import (
	"dataproviders"
	"httpclient"
	"testing"
	"time"
)

type PlantStatsStore struct {
}

func (p PlantStatsStore) LoadStats(plantkey string) dataproviders.PlantStats {
	return dataproviders.PlantStats{}
}
func (p PlantStatsStore) SaveStats(plantkey string, pv *dataproviders.PvData) {}

func Test_Connection(t *testing.T) {
	t.Log("Trying to launch provider...")

	initiateData := dataproviders.InitiateData{"peterlarsen", "jesper@jbr.dk", "cidaxura", "3", ""}

	sunny := sunnyDataProvider{initiateData,
		nil,
		httpclient.NewClient(),
		"",
		""}

	t.Log("Prelogin...")
	if err := sunny.preLogin(); err != nil {
		t.Error(err.Error())
		return
	}
	t.Log("Login...")
	if err := sunny.login(initiateData.UserName, initiateData.Password); err != nil {
		t.Error(err.Error())
		return
	}

	plantname, err := sunny.plantName()
	if err != nil {
		t.Error(err.Error())
		return
	}
	t.Log("Plantname before switching is ", plantname)

	if err := sunny.setPlantNo(initiateData.PlantNo); err != nil {
		t.Error(err.Error())
		return
	}

	plantname, _ = sunny.plantName()

	if plantname == "" || plantname != "LilleStoreLarsen" {
		t.Errorf("Incorrect plant name recieved from sunnyportal = '%s'", plantname)
		return
	}
	t.Log("Waiting 15 secs for the connection to come up...")
	time.Sleep(10 * time.Second)

	pac, err := updatePacData(sunny.client)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Pac data is %d", pac)

	pvdaily, err := updateDailyProduction(sunny.client)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Dailyproduction data is %d", pvdaily)

}
