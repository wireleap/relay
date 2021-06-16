// Copyright (c) 2021 Wireleap

package checkconfigcmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/relaycfg"
)

var Cmd = &cli.Subcmd{
	FlagSet: flag.NewFlagSet("check-config", flag.ExitOnError),
	Desc:    "Validate wireleap-relay config file",
	Run: func(fm fsdir.T) {
		c := relaycfg.Defaults()
		if err := fm.Get(&c, filenames.Config); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := c.Validate(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("OK")
	},
}
