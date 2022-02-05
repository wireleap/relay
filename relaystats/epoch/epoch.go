// Copyright (c) 2022 Wireleap

package epoch

import (
	"time"
)

func EpochMillis() int64 {
	return ToEpochMillis(time.Now())
}

func ToEpochMillis(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

func FromEpochMillis(i int64) time.Time {
	return time.Unix(0, i*1000000)
}
