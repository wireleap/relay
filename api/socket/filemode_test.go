package socket

import (
	"testing"
)

/**
func TestBaseConversion(t *testing.T) {
	result := baseConversion(10, 10, 8)
	if result != uint32(12) {
		t.Fatal("Invalid result")
	}

	result = baseConversion(20, 8, 10)
	if result != uint32(16) {
		t.Fatal("Invalid result")
	}
}

func TestValidateBase(t *testing.T) {
	err := validateBase(0, 11)
	if err == nil {
		t.Fatal("Must have failed")
	}

	err = validateBase(2, 2)
	if err == nil {
		t.Fatal("Must have failed")
	}

	err = validateBase(3, 4)
	if err != nil {
		t.Fatal(err)
	}
}
**/

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
