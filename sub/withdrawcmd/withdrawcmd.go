// Copyright (c) 2021 Wireleap

package withdrawcmd

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/wireleap/common/api/auth"
	"github.com/wireleap/common/api/client"
	"github.com/wireleap/common/api/signer"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
	"github.com/wireleap/relay/relaylib"
)

func Cmd() *cli.Subcmd {
	var (
		fs          = flag.NewFlagSet("withdraw", flag.ExitOnError)
		contract    = fs.String("contract", "", "Service contract URL")
		amount      = fs.String("amount", "", "Withdraw given amount from the balance")
		destination = fs.String("destination", "", "Withdraw to this destination")
	)

	run := func(fm fsdir.T) {
		c := relaycfg.Defaults()
		err := fm.Get(&c, filenames.Config)

		if err != nil {
			log.Fatal(err)
		}

		switch {
		case *contract == "":
			log.Fatal("contract has to be set")
		case *destination == "":
			log.Fatal("destination has to be set")
		case *amount == "":
			log.Fatal("amount has to be set")
		}

		privkey, err := cli.LoadKey(fm, filenames.Seed)

		if err != nil {
			log.Fatal(err)
		}

		var (
			s  = signer.New(privkey)
			cl = client.New(s, auth.Relay)
		)

		amt, err := strconv.ParseInt(*amount, 10, 64)

		if err != nil {
			log.Fatal(err)
		}

		w, err := relaylib.Withdraw(
			cl,
			*contract+"/payout/withdrawals",
			*destination,
			amt,
		)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(*w))
	}

	r := &cli.Subcmd{
		FlagSet: fs,
		Desc:    "Withdraw available funds from balance",
		Run:     run,
	}

	return r
}
