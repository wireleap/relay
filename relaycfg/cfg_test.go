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

func TestCfgRestapiSocket(t *testing.T) {
	b, err := ioutil.ReadFile("../testdata/restapi/config-socket-backing.json")

	if err != nil {
		t.Fatal(err)
	}

	c := Defaults()
	err = json.Unmarshal(b, &c)

	if err != nil {
		t.Fatal(err)
	}

	if c.RestApi.Address.Scheme != "file" {
		t.Fatal("wrong host")
	} else if c.RestApi.Address.Host != "" {
		t.Fatal("wrong host")
	} else if c.RestApi.Address.Path != "/somepath.ext" {
		t.Fatal("wrong path")
	} else if c.RestApi.Umask != 0123 {
		t.Fatal("wrong permissions")
	}
}

func TestCfgRestapi2(t *testing.T) {
	b, err := ioutil.ReadFile("../testdata/restapi/config-port-entropic.json")

	if err != nil {
		t.Fatal(err)
	}

	c := Defaults()
	err = json.Unmarshal(b, &c)

	if err != nil {
		t.Fatal(err)
	}

	if c.RestApi.Address.Scheme != "http" {
		t.Fatal("wrong host")
	} else if c.RestApi.Address.Host != "0.0.0.0:8080" {
		t.Fatal("wrong host")
	} else if c.RestApi.Address.Path != "" {
		t.Fatal("wrong path")
	} else if c.RestApi.Umask != 0600 { // Default permissions
		t.Fatal("wrong permissions")
	}
}
