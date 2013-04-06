package sunnyportal

import (
	"testing"
	"httpclient"
	"dataproviders"
	"time"
	
	
)

func Test_Connection (t *testing.T) {
	t.Log("Trying to launch provider...")
	p, err := NewDataProvider(dataproviders.InitiateData{"jesper@jbr.dk", "cidaxura", "3"}, 
		func(){}, httpclient.NewClient())
		
	if err != nil {
		t.Error(err.Error())
	}
	
	t.Log("Waiting 5 secs for the connection to come up...");
	time.Sleep( 5 * time.Second);
	
	plantname, _ := p.plantName()
	
	if plantname == "" || plantname != "LilleStoreLarsen" {
		t.Errorf("Incorrect plant name recieved from sunnyportal = '%s'", plantname)
	}
	
	pac, err := updatePacData(p.client)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("Pac data is %d", pac)

}

