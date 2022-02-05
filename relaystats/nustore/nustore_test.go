// Copyright (c) 2022 Wireleap

package nustore

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test(t *testing.T) {
	tmpd, err := ioutil.TempDir("", "nutest.*")

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpd)
	})

	s, err := New(tmpd)

	m := map[string]uint64{
		"ct1": uint64(100),
		"ct2": uint64(200),
		"ct3": uint64(300),
	}

	since := int64(100)
	until := int64(200)

	archive := NewArchiveFile("someRelayId", map[string]bool{}, m, since, until)

	if err := s.Add(archive); err != nil {
		t.Fatalf("error should ne nil: %s", err)
	}
}
