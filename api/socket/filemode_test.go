package socket

import (
	"strconv"
	"testing"
)

func TestValidateBase(t *testing.T) {
	i, err := strconv.ParseUint("777", 8, 9)
	if err != nil {
		t.Fatal(err)
	} else if i != 0777 {
		t.Fatal("Value wrongly parsed")
	}

	_, err = strconv.ParseUint("002", 2, 9)
	if err == nil {
		t.Fatal("Must have failed")
	}

	i, err = strconv.ParseUint("003", 4, 9)
	if err != nil {
		t.Fatal(err)
	} else if i != 3 {
		t.Fatal("Value wrongly parsed")
	}

	_, err = strconv.ParseUint("1000", 8, 9)
	if err == nil {
		t.Fatal("Must have failed")
	}

	_, err = strconv.ParseUint("1000", 8, 10)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileModeUnmarshal(t *testing.T) {
	var fm FileMode

	err := fm.UnmarshalJSON([]byte(`"000"`))
	if err != nil {
		t.Fatal(err)
	} else if fm != 0 {
		t.Fatal("Wrong value")
	}

	err = fm.UnmarshalJSON([]byte(`"0000"`))
	if err == nil {
		t.Fatal("Must have failed")
	}

	err = fm.UnmarshalJSON([]byte(`"abc"`))
	if err == nil {
		t.Fatal("Must have failed")
	}

	err = fm.UnmarshalJSON([]byte(`"800"`))
	if err == nil {
		t.Fatal("Must have failed")
	}

	err = fm.UnmarshalJSON([]byte(`"066"`))
	if err != nil {
		t.Fatal(err)
	} else if fm != 54 {
		t.Fatal("Wrong value")
	}
}

func TestFileModeMarshal(t *testing.T) {
	var fm FileMode

	value, err := fm.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	} else if string(value) != `"000"` {
		t.Fatal("Wrong value")
	}

	fm = 0600
	value, err = fm.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	} else if string(value) != `"600"` {
		t.Fatal("Wrong value")
	}

	fm = 06000
	value, err = fm.MarshalJSON()
	if err == nil {
		t.Fatal("Must have failed")
	}
}
