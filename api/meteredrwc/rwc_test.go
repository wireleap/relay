// Copyright (c) 2022 Wireleap

package meteredrwc

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

var test = []byte{'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd'}

const bufsize int = 2048

func TestRetransmit(t *testing.T) {
	ec := make(chan error)
	var i uint64

	c1, c2 := net.Pipe()
	go c2.Write(test) // Buffer alike

	r := New(c1, &i)

	var w bytes.Buffer

	go func() {
		buf := make([]byte, bufsize)
		_, err := io.CopyBuffer(&w, r, buf)
		ec <- err
	}()

	time.Sleep(time.Second)
	r.Close()
	<-ec
}
