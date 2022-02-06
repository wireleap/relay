// Copyright (c) 2022 Wireleap

package relayentryext

import (
	"github.com/wireleap/common/api/relayentry"

	"github.com/c2h5oh/datasize"
)

type T struct {
	// Common relayentry
	relayentry.T
	// Network usage limit
	NetUsage datasize.ByteSize `json:"network_usage_limit,omitempty"`
}

func (r *T) String() string {
	return r.T.String()
}

func (r *T) Validate() error {
	return r.T.Validate()
}
