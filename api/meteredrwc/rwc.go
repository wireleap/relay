// Copyright (c) 2022 Wireleap

package meteredrwc

import (
	"io"
	"sync/atomic"
	"time"
)

/**
  Updates:
  - synced *uint64 + internal int on read
  - prometheus telemetry (duration + bytes) on close [ToDo]
**/
type MRWC struct {
	rwc   io.ReadWriteCloser
	bytes int
	//promCounter prometheus.Counter
	//promHist TimeHistogram
	startAt   time.Time
	syncBytes *uint64
}

func New(rwc io.ReadWriteCloser, syncBytes *uint64) io.ReadWriteCloser {
	// ToDo: Add promCounter and promHist
	return &MRWC{
		rwc:       rwc,
		startAt:   time.Now(),
		syncBytes: syncBytes,
	}
}

func (mRWC *MRWC) update(i int) {
	if mRWC.syncBytes != nil {
		atomic.AddUint64(mRWC.syncBytes, uint64(i))
	}
	mRWC.bytes = mRWC.bytes + i
}

func (mRWC *MRWC) Read(p []byte) (n int, err error) {
	n, err = mRWC.rwc.Read(p)
	mRWC.update(n)
	return
}

func (mRWC *MRWC) Write(p []byte) (n int, err error) {
	return mRWC.rwc.Write(p)
}

func (mRWC *MRWC) Close() error {
	//duration := time.Since(mRWC.startAt)
	return mRWC.rwc.Close()
}
