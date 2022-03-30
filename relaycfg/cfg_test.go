// Copyright (c) 2022 Wireleap

package relaycfg

import (
	"encoding/json"
	"fmt"
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

func TestCfgNetwork(t *testing.T) {
	b, err := ioutil.ReadFile("../testdata/network/config-backing-global-contract.json")

	if err != nil {
		t.Fatal(err)
	}

	c := Defaults()
	err = json.Unmarshal(b, &c)

	if err != nil {
		t.Fatal(err)
	}

	for _, v := range c.Contracts {
		fmt.Println(v)
		fmt.Println(v.NetUsage)
	}
}
