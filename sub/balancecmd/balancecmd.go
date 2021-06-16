// Copyright (c) 2021 Wireleap

package balancecmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/wireleap/common/api/auth"
	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/api/texturl"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
)

func Cmd() *cli.Subcmd {
	c := relaycfg.Defaults()

	fs := flag.NewFlagSet("balance", flag.ExitOnError)
	contract := fs.String("contract", "", "Service contract URL")

	run := func(fm fsdir.T) {
		err := fm.Get(&c, filenames.Config)

		if err != nil {
			log.Fatal(err)
		}

		scs := []*texturl.URL{}

		type Results struct {
			Contract   string           `json:"contract"`
			Balance    *json.RawMessage `json:"balance"`
			Tokens     *json.RawMessage `json:"tokens"`
			Withdrawal *json.RawMessage `json:"withdrawal"`
		}

		switch {
		case *contract == "":
			// all contracts
			log.Println("no contract specified, listing balance for all contracts...")

			for u, _ := range c.Contracts {
				scs = append(scs, &u)
			}
		default:
			scs = []*texturl.URL{texturl.URLMustParse(*contract)}
		}

		privkey, err := cli.LoadKey(fm, filenames.Seed)

		if err != nil {
			log.Fatal(err)
		}

		var (
			s  = signer.New(privkey)
			cl = client.New(s, auth.Relay)
		)

		for _, u := range scs {
			sc := u.String()

			b, err := relaylib.Get(cl, sc+"/payout/balance")

			if err != nil {
				log.Fatalf("error while getting balance: %s", err)
			}

			ts, err := relaylib.Get(cl, sc+"/payout/tokens")

			if err != nil {
				log.Fatalf("error while getting accumulated tokens: %s", err)
			}

			w, err := relaylib.Get(cl, sc+"/payout/withdrawals")

			if err != nil {
				log.Fatalf("error while getting withdrawal: %s", err)
			}

			r := Results{
				Contract:   sc,
				Balance:    b,
				Tokens:     ts,
				Withdrawal: w,
			}

			data, err := json.MarshalIndent(r, "", "    ")

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(data))
		}
	}

	r := &cli.Subcmd{
		FlagSet: fs,
		Desc:    "Show balance, pending sharetokens and last withdrawal",
		Run:     run,
	}

	return r
}
