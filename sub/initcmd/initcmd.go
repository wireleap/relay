// Copyright (c) 2022 Wireleap

package initcmd

import (
	"flag"
	"log"

	"github.com/wireleap/common/api/tlscert"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/commonsub/initcmd"
	"github.com/wireleap/common/cli/fsdir"
	"github.com/wireleap/relay/filenames"
	"github.com/wireleap/relay/sub/initcmd/embedded"
)

var Cmd = &cli.Subcmd{
	FlagSet: flag.NewFlagSet("init", flag.ExitOnError),
	Desc:    "Generate ed25519 keypair and TLS cert/key",
	Run: func(fm fsdir.T) {
		initcmd.Cmd("wireleap-relay", initcmd.KeypairStep, initcmd.UnpackStep(embedded.FS)).Run(fm)
		privkey, err := cli.LoadKey(fm, filenames.Seed)
		if err != nil {
			log.Fatal(err)
		}
		if err := tlscert.Generate(fm.Path(filenames.TLSCert), fm.Path(filenames.TLSKey), privkey); err != nil {
			log.Fatal(err)
		}
	},
}
