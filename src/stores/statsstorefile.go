package stores

import (
	"io/ioutil"
	"dataproviders"
	"encoding/json"
)

const Statfilename = "_stats.json"

type StatsStoreFile struct {
}

// Load up stats from filesystem
func (s StatsStoreFile)LoadStats(plantkey string) dataproviders.PlantStats {
	stats := dataproviders.PlantStats{}
	bytes, err := ioutil.ReadFile(plantkey + Statfilename)
	if err != nil {
		log.Infof("Error in reading statfile for plant %s: %s", plantkey, err.Error())
		return stats
	}
	err = json.Unmarshal(bytes, &stats)
	return stats
}

func (s StatsStoreFile)SaveStats(plantkey string, pv *dataproviders.PvData) {
	stats := dataproviders.PlantStats{}
	stats.PowerAcPeakAll = pv.PowerAcPeakAll
	stats.PowerAcPeakAllTime = pv.PowerAcPeakAllTime
	stats.PowerAcPeakToday = pv.PowerAcPeakToday
	stats.PowerAcPeakTodayTime = pv.PowerAcPeakTodayTime
	bytes, err := json.Marshal(stats)
	if err != nil {
		log.Failf("Could not marshal plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
	err = ioutil.WriteFile(plantkey+Statfilename, bytes, 0777)
	if err != nil {
		log.Failf("Could not write plant stats for plant %s: %s", plantkey, err.Error())
		return
	}
}

