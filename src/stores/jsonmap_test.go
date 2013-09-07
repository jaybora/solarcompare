package stores

import (
	"encoding/json"
	"testing"
)

type something struct {
	Astring string
	Anumber int
}

func Test_Map(t *testing.T) {
	m := make(map[string]something)
	m["noget"] = something{"the string", 100}
	m["nogetandet"] = something{"the secound string", 10}

	b, err := json.MarshalIndent(&m, "", "  ")

	if err != nil {
		t.Errorf("Error %s", err.Error())
	}

	t.Logf("JSON is \n %s", b)

}
