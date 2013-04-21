package dataproviders

import (
	"testing"
	"time"

)

/*
To run test export GOPATH=/Users/jbr/github/local/solarcompare
then
go test -test.v dataproviders
*/

func Test_peak_startset (t *testing.T) {
	
	//Make channels
	reqCh := make(chan chan PvData)
	updateCh := make(chan PvData)
	terminateCh := make(chan int)

	go LatestPvData(reqCh, updateCh, terminateCh, func(plantkey string, pv PvData){}, "key")
	
	
	pv := PvData{}
	pv.PowerAcPeakAll = 100
	pv.PowerAcPeakAllTime = time.Now()
	pv.PowerAcPeakToday = 101
	pv.PowerAcPeakTodayTime = time.Now()

	updateCh <- pv
	
	// Get pv data out from LatestPvData again
	pvCh := make(chan PvData)
	reqCh <- pvCh
	pv = <-pvCh
	
	if pv.PowerAcPeakAll != 100 {
		t.Errorf("PowerAcPeakAll should be 100 was %d", pv.PowerAcPeakAll)
	}
	if pv.PowerAcPeakToday != 101 {
		t.Errorf("PowerAcPeakToday should be 101 was %d", pv.PowerAcPeakToday)
	}
	
}
func Test_peak (t *testing.T) {
	
	//Make channels
	reqCh := make(chan chan PvData)
	updateCh := make(chan PvData)
	terminateCh := make(chan int)

	go LatestPvData(reqCh, updateCh, terminateCh, func(plantkey string, pv PvData){}, "key")
	
	now:=time.Now()
	updateCh <- PvData{LatestUpdate: &now, PowerAc: 100}
	
	// Get pv data out from LatestPvData again
	pvCh := make(chan PvData)
	reqCh <- pvCh
	pv := <-pvCh
	
	if pv.PowerAcPeakAll != 100 {
		t.Errorf("PowerAcPeakAll should be 100 was %d", pv.PowerAcPeakAll)
	}

	if pv.PowerAcPeakToday != 100 {
		t.Errorf("PowerAcPeakAll should be 100 was %d", pv.PowerAcPeakToday)
	}

	if !pv.PowerAcPeakAllTime.Equal(now) {
		t.Errorf("PowerAcPeakAllTime should be %s was %s", now, pv.PowerAcPeakAllTime)
	}
	
	if !pv.PowerAcPeakTodayTime.Equal(now) {
		t.Errorf("PowerAcPeakTodayTime should be %s was %s", now, pv.PowerAcPeakTodayTime)
	}
	
	time.Sleep(1 * time.Second)
	
	// Update again
	now = time.Now()
	updateCh <- PvData{LatestUpdate: &now, PowerAc: 200}
	time.Sleep(1 * time.Second)

	// Get pv data out from LatestPvData again
	pvCh = make(chan PvData)
	reqCh <- pvCh
	pv = <-pvCh

	if pv.PowerAcPeakAll != 200 {
		t.Errorf("PowerAcPeakAll should be 200 was %d", pv.PowerAcPeakAll)
	}

	if pv.PowerAcPeakToday != 200 {
		t.Errorf("PowerAcPeakAll should be 200 was %d", pv.PowerAcPeakToday)
	}

	if !pv.PowerAcPeakAllTime.Equal(now) {
		t.Errorf("PowerAcPeakAllTime should be %s was %s", now, pv.PowerAcPeakAllTime)
	}
	
	if !pv.PowerAcPeakTodayTime.Equal(now) {
		t.Errorf("PowerAcPeakTodayTime should be %s was %s", now, pv.PowerAcPeakTodayTime)
	}
	
	// Ok lower again, we should see the same result again
	time.Sleep(1 * time.Second)
	
	// Update again
	newnow := time.Now()
	updateCh <- PvData{LatestUpdate: &newnow, PowerAc: 198}
	time.Sleep(1 * time.Second)

	// Get pv data out from LatestPvData again
	pvCh = make(chan PvData)
	reqCh <- pvCh
	pv = <-pvCh

	if pv.PowerAcPeakAll != 200 {
		t.Errorf("PowerAcPeakAll should be 200 was %d", pv.PowerAcPeakAll)
	}

	if pv.PowerAcPeakToday != 200 {
		t.Errorf("PowerAcPeakToday should be 200 was %d", pv.PowerAcPeakToday)
	}

	if !pv.PowerAcPeakAllTime.Equal(now) {
		t.Errorf("PowerAcPeakAllTime should be %s was %s", now, pv.PowerAcPeakAllTime)
	}
	
	if !pv.PowerAcPeakTodayTime.Equal(now) {
		t.Errorf("PowerAcPeakTodayTime should be %s was %s", now, pv.PowerAcPeakTodayTime)
	}
	
	// Check for falling PowerAcPeak should not be stored
	updateCh <- PvData{}
	// Get pv data out from LatestPvData again
	pvCh = make(chan PvData)
	reqCh <- pvCh
	pv = <-pvCh

	if pv.PowerAcPeakAll != 200 {
		t.Errorf("PowerAcPeakAll should be 200 was %d", pv.PowerAcPeakAll)
	}
	
	// Check for raising PowerAcPeak should be stored
	pv.PowerAcPeakAll = 251
	pv.PowerAcPeakToday = 250
	updateCh <- pv
	// Get pv data out from LatestPvData again
	pvCh = make(chan PvData)
	reqCh <- pvCh
	pv = <-pvCh

	if pv.PowerAcPeakAll != 251 {
		t.Errorf("PowerAcPeakAll should be 250 was %d", pv.PowerAcPeakAll)
	}
	if pv.PowerAcPeakToday != 250 {
		t.Errorf("PowerAcPeakToday should be 250 was %d", pv.PowerAcPeakToday)
	}
		
	return
}

