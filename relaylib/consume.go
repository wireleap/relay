// Copyright (c) 2022 Wireleap

package relaylib

import (
	"fmt"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/consume"
	"github.com/wireleap/common/api/jsonb"
	"github.com/wireleap/common/api/texturl"
)

// Get contract PubKey
func getPK(cl *client.Client, contractUrl texturl.URL) (string, error) {
	pk, err := consume.ContractPubkey(cl, &contractUrl)

	if err != nil {
		return "", fmt.Errorf("could not get pubkey for %s: %s", contractUrl.String(), err)
	}

	return jsonb.PK(pk).String(), nil
}
