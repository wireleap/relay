// Copyright (c) 2021 Wireleap

package relaycfg

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestCfg(t *testing.T) {
	b, err := ioutil.ReadFile("../testdata/config-backing.json")

	if err != nil {
		t.Fatal(err)
	}

	c := Defaults()
	err = json.Unmarshal(b, &c)

	if err != nil {
		t.Fatal(err)
	}

	b, err = ioutil.ReadFile("../testdata/config-backing.json")

	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(b, &c)

	if err != nil {
		t.Fatal(err)
	}
}
