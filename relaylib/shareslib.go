// Copyright (c) 2022 Wireleap

package relaylib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/sharetoken"
	"github.com/wireleap/common/api/status"
	"github.com/wireleap/common/api/withdrawalrequest"
)

func SubmitST(cl *client.Client, url string, st *sharetoken.T) error {
	log.Printf(
		"submitting sharetoken sig=%s to %s (%s)",
		st.Signature,
		url,
		st.Contract.PublicKey,
	)

	ret := &json.RawMessage{}
	err := cl.Perform(http.MethodPost, url, st, ret)

	if err != nil {
		return err
	}

	return nil
}

func SubmitSTFromFile(cl *client.Client, url, shares string) error {
	var ts []*sharetoken.T

	rdata, err := ioutil.ReadFile(shares)

	if err != nil {
		return err
	}

	err = json.Unmarshal(rdata, &ts)

	if err != nil {
		return err
	}

	var left []*sharetoken.T
	var errs []error

	for _, st := range ts {
		err = SubmitST(cl, url, st)

		if err != nil {
			if errors.Is(err, status.ErrSettlementClosed) {
				log.Printf(
					"discarding sharetoken submitted post-settlement: %s",
					st.PublicKey,
				)

				continue
			}

			// submit as many STs as possible
			left = append(left, st)
			errs = append(errs, err)
		}
	}

	wdata, err := json.Marshal(left)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(shares, wdata, 0644)

	if err != nil {
		return err
	}

	if len(left) > 0 {
		type leftover struct{ pubkey, error string }

		var leftovers []leftover

		for i, st := range left {
			pk := st.PublicKey.String()
			leftovers = append(leftovers, leftover{pk, errs[i].Error()})
		}

		return fmt.Errorf("these sharetokens were not submitted because of errors: %+v", leftovers)
	}

	proc, _ := os.FindProcess(os.Getpid())
	err = proc.Signal(syscall.SIGUSR1)

	if err != nil {
		return fmt.Errorf("error sending SIGUSR1 to self! %w", err)
	}

	return nil
}

func Withdraw(cl *client.Client, url string, destination string, amount int64) (*json.RawMessage, error) {
	w := &withdrawalrequest.T{
		Amount:      amount,
		Type:        "TODO", // TODO currently unused
		Destination: destination,
	}

	err := w.Validate()

	if err != nil {
		return nil, err
	}

	// double indirection so we can treat "null" as nil
	ret := &json.RawMessage{}
	err = cl.Perform(http.MethodPost, url, w, ret)

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func Get(cl *client.Client, url string) (*json.RawMessage, error) {
	// double indirection so we can treat "null" as nil
	ret := &json.RawMessage{}
	err := cl.Perform(http.MethodGet, url, nil, &ret)

	if err != nil {
		return nil, err
	}

	return ret, nil
}
