// Copyright (c) 2021 Wireleap

package main

import (
	"os"

	"github.com/wireleap/common/api/interfaces/clientrelay"
	"github.com/wireleap/common/api/interfaces/relaycontract"
	"github.com/wireleap/common/api/interfaces/relaydir"
	"github.com/wireleap/common/api/interfaces/relayrelay"
	"github.com/wireleap/common/cli"
	"github.com/wireleap/common/cli/commonsub/commonlib"
	"github.com/wireleap/common/cli/commonsub/migratecmd"
	"github.com/wireleap/common/cli/commonsub/reloadcmd"
	"github.com/wireleap/common/cli/commonsub/restartcmd"
	"github.com/wireleap/common/cli/commonsub/rollbackcmd"
	"github.com/wireleap/common/cli/commonsub/statuscmd"
	"github.com/wireleap/common/cli/commonsub/stopcmd"
	"github.com/wireleap/common/cli/commonsub/superviseupgradecmd"
	"github.com/wireleap/common/cli/commonsub/upgradecmd"
	"github.com/wireleap/common/cli/commonsub/versioncmd"
	"github.com/wireleap/common/cli/upgrade"

	"github.com/wireleap/relay/sub/balancecmd"
	"github.com/wireleap/relay/sub/checkconfigcmd"
	"github.com/wireleap/relay/sub/initcmd"
	"github.com/wireleap/relay/sub/startcmd"
	"github.com/wireleap/relay/sub/withdrawcmd"
	"github.com/wireleap/relay/version"
)

const binname = "wireleap-relay"

func main() {
	cli.CLI{
		Subcmds: []*cli.Subcmd{
			initcmd.Cmd,
			startcmd.Cmd(),
			stopcmd.Cmd(binname),
			restartcmd.Cmd(binname, startcmd.Cmd().Run, stopcmd.Cmd(binname).Run),
			reloadcmd.Cmd(binname),
			statuscmd.Cmd(binname),
			upgradecmd.Cmd(
				binname,
				upgrade.ExecutorSupervised,
				version.VERSION,
				version.LatestChannelVersion,
			),
			superviseupgradecmd.Cmd(commonlib.Context{
				BinName:    binname,
				NewVersion: version.VERSION,
			}),
			migratecmd.Cmd(binname, version.MIGRATIONS, version.VERSION),
			rollbackcmd.Cmd(commonlib.Context{
				BinName:  binname,
				PostHook: version.PostRollbackHook,
			}),
			checkconfigcmd.Cmd,
			balancecmd.Cmd(),
			withdrawcmd.Cmd(),
			versioncmd.Cmd(
				&version.VERSION,
				relaycontract.T,
				relaydir.T,
				relayrelay.T,
				clientrelay.T,
			),
		},
	}.Parse(os.Args).Run(cli.Home())
}
