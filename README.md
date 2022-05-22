# Wireleap relay

[Wireleap](https://wireleap.com) is a decentralized communications protocol
and open-source software designed with the goal of providing unrestricted
access to the internet from anywhere.

The Wireleap relay software is used to relay Wireleap protocol traffic from
clients and other relays.

This repository is for the Wireleap relay.

## Table of contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Web server proxying](#web-server-proxying)
    - [Protocol encapsulation](#protocol-encapsulation)
    - [Fronting relay configuration example](#fronting-relay-configuration-example)
    - [Apache configuration example](#apache-configuration-example)
    - [Nginx configuration example](#nginx-configuration-example)
- [Network usage](#network-usage)
    - [Network usage measurement](#network-usage-measurement)
    - [Network cap](#network-cap)
      - [Thresholds](#thresholds)
    - [Network usage configuration example](#network-usage-configuration-example)
- [POSIX signal hooks](#posix-signal-hooks)
- [Testing](#testing)
- [Production](#production)
    - [Increase ulimit](#increase-ulimit)
    - [Daemon supervisor](#daemon-supervisor)
- [Settlement](#settlement)
    - [Submitting sharetokens](#submitting-sharetokens)
    - [Checking status](#checking-status)
    - [Initiating a withdrawal request](#initiating-a-withdrawal-request)
- [Upgrade](#upgrade)
- [Versioning](#versioning)
- [Building](#building)
- [Contributing](#contributing)
    - [Fork, clone and setup upstream remote](#fork-clone-and-setup-upstream-remote)
    - [Create a feature branch and make your changes](#create-a-feature-branch-and-make-your-changes)
    - [Unit testing](#unit-testing)
    - [Rebase on master if needed](#rebase-on-master-if-needed)
    - [Push changes and submit a pull request](#push-changes-and-submit-a-pull-request)
    - [Review process and merge](#review-process-and-merge)
- [License](#license)

## Installation

The recommended installation procedure starts with the creation of a
system user account specifically for the wireleap-relay.

```shell
useradd -rUm --home-dir /opt/wireleap-relay wireleap-relay
```

Next, download and verify the [latest release][releases]. Alternatively,
you can [build from source](#building).

[releases]: https://github.com/wireleap/relay/releases

```shell
su -l wireleap-relay

# download binary and hashfile for Linux
DIST="https://github.com/wireleap/relay/releases/latest/download"
curl -O $DIST/wireleap-relay_linux-amd64
curl -O $DIST/wireleap-relay_linux-amd64.hash

# cryptographically verify integrity of the hashfile
gpg --recv-keys 693C86E9DECA9D07D79FF9D22ECD72AD056012E1
gpg --list-keys --with-fingerprint builds@wireleap.com
gpg --verify wireleap-relay_linux-amd64.hash

# verify checksum hash
sha512sum -c wireleap-relay_linux-amd64.hash

# rename the binary and set executable flag
mv wireleap-relay_linux-amd64 wireleap-relay
chmod +x wireleap-relay

# generate certificates and a keypair specific for the relay
./wireleap-relay init

# create a configuration file (see the next section)
$EDITOR config.json
```

Once configured, proceed to the [testing](#testing) and/or
[production](#production) section.

## Configuration

Relays are configured through the file `config.json`, which includes
service contracts the relay supports along with a specific configuration
per each contract as well as the daemon configuration. Currently
supported variables:

Key | Type | Comment
--- | ---- | -------
address | `string` | address to bind to (`host:port`)
archive_dir | `string` | path to archive submitted sharetokens (optional)
auto_submit_interval | `string` | interval between sharetoken submission retries (optional)
network_usage.global_limit | `string` | maximum routed traffic in defined period (optional)
network_usage.timeframe | `string` | routed traffic measurement fixed time window (optional)
network_usage.write_interval | `string` | interval between autosaves (optional)
network_usage.archive_dir | `string` | path of the archived statistics directory (optional)
contracts.X | `string` | service contract endpoint url
contracts.X.address | `string` | `wireleap://host:port[/uri]`
contracts.X.role | `string` | `fronting` `entropic` `backing`
contracts.X.key | `string` | `user:password` format enrollment key if required
contracts.X.network_usage_limit | `string` | maximum routed traffic for this contract
contracts.X.upgrade_channel | `string` | upgrade channel (default: `"default"`)
auto_upgrade | `bool` | automatically upgrade this relay (default: `true`)

```json
{
    "address": "0.0.0.0:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "network_usage": {
        "global_limit": "2TB",
        "timeframe": "30d",
        "write_interval": "5m0s",
        "archive_dir": "archive/netstats"
    },
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "backing",
            "key": "backing:secretkey"
        },
        "https://contract2.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "entropic"
        },
        "https://contract3.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "entropic",
            "network_usage_limit": "1TB"
        }
    }
}
```

Note: A `fronting` relay generally requires a [webserver
proxying](#web-server-proxying) configuration.

## Web server proxying

When the `fronting` relay daemon is behind a proxying web server that
supports HTTP/2, such as Apache or Nginx, the web server can be
configured to proxy connections to the daemon while simultaneously
serving the regular website traffic.

### Protocol encapsulation

A HTTP/2 connection is used as transport and the individual connections
are streams inside it. The HTTP/2 connection is opaque and valid traffic
for the webserver to proxy. Once the proxying webserver has performed
the TLS and H/2 negotiation (settings frame, etc.) and bidirectional
streaming data transfer is setup, the encapsulated traffic is seamlessly
proxied to and from the relay daemon `address`.

### Fronting relay configuration example

```json
{
    "address": "127.0.0.1:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://www.example.com:443/wireleap",
            "role": "fronting"
        }
    }
}
```

### Apache configuration example

```apache
<IfModule ssl_module>
 <IfModule http2_module>
   Listen 443 https
   Protocols h2
   <VirtualHost *:443>
     ServerAdmin admin@example.com
     SSLCertificateFile /path/to/crt
     SSLCertificateKeyFile /path/to/key
     SSLEngine on
     SSLProxyEngine on
     SSLProxyCheckPeerName off
     DocumentRoot /var/www/html

     <Location "/wireleap">
       <IfModule mod_reqtimeout.c>
         RequestReadTimeout handshake=0 header=0 body=0
       </IfModule>
       ProxyPass "h2://127.0.0.1:13490"
       ProxyPassReverse "h2://127.0.0.1:13490"
     </Location>
   </VirtualHost>
 </IfModule>
</IfModule>
```

### Nginx configuration example

```nginx
server {
    listen 443 ssl http2;
    server_name www.example.com;
    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;
    ssl_protocols TLSv1.3;

    location /wireleap {
        grpc_pass grpcs://127.0.0.1:13490;
        grpc_ssl_protocols TLSv1.3;
    }
}
```

## Network usage

Network usage feature enables the operators to monitor and limit the amount of
data routed through the relay, and for each contract. This feature includes
status storage accross application restarts and generates historical records of
each period.

### Network usage measurement

Network traffic is a key and limited resource, just like CPU and RAM.
Sadly, measuring accurately the used traffic at an application level on modern
languages is not an easy task Currently the application records the size of the
routed TCP streams. H/2, TCP, IP, and lower headers are not taken into account.

Ignoring the H/2 headers, the remaining headers are: TCP (layer 4) and IP (layer
3). Lower layers aren't taken into account because they belong to the local
area network or physical point to point link scopes. In other words, they're
not routed though the Internet.

Each IP packet has a header and a payload. The payload is a TCP packet, which
also has a header and a payload. The TCP payload is, overall, what we're
currently capable of measuring.

Depending on the length of the IP packets, the measured traffic value will
deviate more or less from the real value. The IP packet length is as low as the
contained payload and as big as the available MTU (Maximum, Transmision Unit);
once hit the payload is sent on multiple IP packets, each one containing also
a TCP header. The IPv4 header size is 20-24 bytes and the TCP one is 20 bytes.
The network MTU usually is 1492-1500 bytes.

For our calculations, we're taking the most common values:
 - IP header: 20 bytes
 - TCP header: 20 bytes
 - MTU: 1492 bytes

The only remaining variable is the `average packet length`, which depends
directly on the the protocols and applications used by the client. Packet size
distribution on computer networks is a quite common paper topic. Sadly, 1)
there's no generic network usage or applications and 2) the nature of the usage
evolves with the time. As a result, we only can take inspiration from those
papers and guess.

Fact 1: Small packets (100 bytes) are commonly used for signalisation purposes,
audio streams or multiplayer video games.
Fact 2: Big packages, hitting the MTU are used for heavy payloads requiring
multiple packets.

Guess: 30% of the packets are small, and 70% are big; the remaining sizes are
marginal. Average packet size is ~1075 bytes, and measured traffic is 95.8%.

| Avg pkt size (bytes) | Traffic measure accuracy    |
| -------------------- | --------------------------- |
| X                    | = (X - headers(IP+TCP)) / X |
| 1500                 | 97.33%                      |
| 1492                 | 97.32%                      |
| 1300                 | 96.92%                      |
| 1100                 | 96.36%                      |
| 900                  | 95.56%                      |
| 700                  | 94.29%                      |
| 500                  | 92%                         |
| 300                  | 86.67%                      |
| 100                  | 60%                         |
| 40                   | 0%                          |

To be sure we're not guessing short, the traffic limitation features activate
at **90%** of the defined threshold and stop all kind of traffic at **93%**.

Finally, it's important to mention that any other kind of traffic is not
currently measured:
- Contract enrollments, disenrollments & heartbeats
- Upgrade downloads
- Sharetoken submissions
- Application telemetry, if any
- Additional traffic result of operator interaction or automations

We consider that traffic to be comparatively marginal, but it might be not.

### Network cap

The network cap feature limits the network traffic routed by the relay.
It supports global and per contract limits. Once a limit is reached, the
relay disenrolls from the affected contract, or from all of them.
Once the current measurement period ends, the relay reconnects to the
defined contracts.

This mechanism is described in detail in the following section.

#### Thresholds

Two thresholds have been set to smooth the transition of the current clients to
other relays: The `soft-limit` is oriented to prevent new clients from
reaching the relay (by refusing new connections and disenrolling the relay
from the contract), whereas the `hard-limit` also closes the remaining active
connections.

Currently the `soft-limit` is activated when reached the **90%** of the "limit
value" of a contract and the `hard-limit` at **93%** of the same value.

It's important to mention that the global limit doesn't support `soft-limit`,
and if reached the relay disconnects all clients and disenrolls from all the 
contracts.

If the relay is connected to only one contract, please also set the limit on the
contract configuration to enable the `soft-limit` feature.

### Network usage configuration example

```json
{
    "address": "127.0.0.1:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "network_usage": {
        "global_limit": "1TB",
        "timeframe": "30d",
        "write_interval": "5m0s",
        "archive_dir": "archive/netstats"
    },
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "entropic",
            "network_usage_limit": "1TB"
        }
    }
}
```

## POSIX signal hooks

Currently the wireleap relay lacks an API to be operated.
The most needed operations can be performed by sending a POSIX signal.

| Signal  | Purpose                  |
|---------|--------------------------|
| SIGUSR1 | Reload the configuration |
| SIGUSR2 | Update `stats.json` file |

## Testing

Once configured, you can test that everything works by manually starting
the daemon.

```shell
su -l wireleap-relay

# start the relay in the foreground (ctrl+c to stop)
./wireleap-relay start --fg

# or, in the background
./wireleap-relay start
./wireleap-relay status
cat wireleap-relay.log
./wireleap-relay stop
```

## Production

### Increase ulimit

The default `ulimit -n` value of `1024` on most systems would likely
prove too low for running a production relay. Consider changing it to a
higher value.

```shell
echo 'wireleap-relay soft nofile 65535' >> /etc/security/limits.conf
echo 'wireleap-relay hard nofile 65535' >> /etc/security/limits.conf
```

### Daemon supervisor

To keep the relay process up and running at all times, the use of a
process supervisor like [systemd][systemd] is recommended. The following
is a suitable systemd unit file, which will watch the relay daemon
process and restart it if it fails for any reason.

[systemd]: https://en.wikipedia.org/wiki/Systemd

```systemd
[Unit]
Description=Wireleap relay process
After=multi-user.target
StartLimitIntervalSec=200
StartLimitBurst=5

[Service]
Type=forking
User=wireleap-relay
Group=wireleap-relay
RootDirectory=/opt/wireleap-relay
ExecStartPre=-/wireleap-relay stop
ExecStart=/wireleap-relay start
ExecReload=/wireleap-relay reload
ExecStop=/wireleap-relay stop
Restart=on-failure
RestartSec=10
KillMode=process
BindReadOnlyPaths=/etc/ssl/certs/ca-certificates.crt /etc/resolv.conf
MountAPIVFS=on
ProtectProc=invisible
ProcSubset=pid
PrivateUsers=on
PrivateDevices=on

[Install]
WantedBy=multi-user.target
```

```shell
# create the systemd unit file and enable it
$EDITOR /etc/systemd/system/wireleap-relay.service
systemctl enable /etc/systemd/system/wireleap-relay.service

# start the service, check on its status
systemctl start wireleap-relay.service
systemctl status wireleap-relay.service
```

## Settlement

A service contract defines the service parameters and facilitates
disbursing funds provided by a customer to service providers in
proportion to service provided based on proof of service.

The proof of service in question are the sharetokens accumulated by a
relay when relaying user traffic.

### Submitting sharetokens

Sharetokens are accumulated when relaying user traffic and automatically
submitted to the contract upon servicekey expiration. If for some reason
the submission fails, it will be retried based on the
`auto_submit_interval` as defined in the relay configuration.

Note: Sharetokens need to be submitted to the contract during the
submission window (from servicekey expiration plus
`settlement.submission_window`) as defined by the service contract.

### Checking status

Once sharetokens are submitted for settlement, their status can be
queried with the `balance` command. When the settlement window closes
and all verification checks are complete, the final calculation is
performed. Based on the calculation, the relay's balance will be
credited, also shown per the `balance` command.

Note: the available balance shown is the integer part of the real,
internally stored balance of the relay. This ensures that settlement
always results in a fair assignment of relay shares. It may also lead to
sub-one cent balance increases under certain conditions which will not
affect the available balance immediately but increase the internally
stored balance.

```shell
su -l wireleap-relay

# show available balance, sharetokens awaiting settlement window,
# and last withdrawal from all contracts
./wireleap-relay balance
```

### Initiating a withdrawal request

A relay operator may issue a withdrawal request up to or equal to their
available balance.

```shell
su -l wireleap-relay

# show available balance
./wireleap-relay balance \
    --contract https://contract1.example.com

# request withdrawal
./wireleap-relay withdraw \
    --contract https://contract1.example.com \
    --destination acct_1032D82eZvKYlo2C \
    --amount 150
```

Information regarding the `--destination` can be obtained by visiting
the link as defined in contract's `payout.info`.

```shell
curl -s https://contract1.example.com/info | jq -r '.payout.info'
```

## Upgrade

The precompiled binary of `wireleap-relay` includes both automatic and
manual upgrade functionality. Due to protocol [versioning](#versioning),
it is highly recommended to keep relays up to date.

**Automatic upgrades**

When `auto_upgrade` is set to `true` or not present in the relay
`config.json`, the relay will attempt automatic upgrades whenever it
receives an upgrade notification from the directory during heartbeat on
the upgrade channel specified in the enrollment config. If an upgrade
fails, a best-effort rollback is performed and the affected version is
skipped.

The relay upgrade channels supported by the directory and the respective
latest versions are exposed via the directory's `/info` endpoint.

```json
{
    "auto_upgrade": true,
    "contracts": {
        "https://contract1.example.com": {
            "upgrade_channel": "default"
        }
    }
}
```

**Manual upgrades**

The upgrade process is interactive so you will have the possibility
to accept or decline based on the changelog for the new release version.

If the upgrade was successful, the old binary is not deleted but kept as
`wireleap-relay.prev` for rollback purposes in case issues manifest
post-upgrade.

If the upgrade was not successful, it is possible to skip the faulty
version explicitly.

```shell
su -l wireleap-relay

# perform interactive upgrade
./wireleap-relay upgrade

# rollback if required
./wireleap-relay rollback

# skip upgrades to version 1.2.3
echo "1.2.3" > .skip-upgrade-version
```

## Versioning

Releases are based on [semantic versioning](https://semver.org),
and use the format `MAJOR.MINOR.PATCH`. While the MAJOR version is `0`,
MINOR version bumps are considered MAJOR bumps per the semver spec.

Git tags are used to specify the software version, which are manually
assigned by tagging the relevant changelog entry. Only tagged versions
are CI-built and released after all unit and integration tests have
passed successfully.

Note: Locally built binaries will include a suffix in addition to the
latest tagged version, consisting of the number of commits past the tag
and the abbreviated hash of the HEAD commit.

## Building

Note: If you would like to make changes to the source code, please
following the [contributing](#contributing) instructions instead.

Note: Custom built binaries do not support [upgrade](#upgrade)
functionality.

**Clone the repository**

```shell
git clone https://github.com/wireleap/relay.git
```

**Checkout the latest tagged version**

For locally built binaries to match the latest stable `wireleap`
version, you will need to check out the latest git tag prior to
building as opposed to building from master.

```shell
cd relay
git pull --tags origin master
git checkout $(git describe `git rev-list --tags --max-count=1`)
```

**Build the binary**

It is recommended to build the binary using docker, as described below
which uses the official `golang` docker image.

```shell
# for your host operating system
./contrib/docker/build-bin.sh build/

# for a specific target os (linux / darwin)
TARGET_OS=linux ./contrib/docker/build-bin.sh build/

# specify a cache for faster subsequent builds
mkdir -p build/.deps
DEPS_CACHE=build/.deps ./contrib/docker/build-bin.sh build/
```

If you prefer to use your host system instead of docker, you can do so
with `contrib/build-bin.sh` provided you have the relevant dependencies
installed.

## Contributing

This flow is loosely based on the standard [GitHub flow][github_flow]
collaborative development model.

Collaboration between developers is facilitated via pull requests from
topic branches towards the `master` branch, and pull request reviews are
used to achieve consensus before merging the changes into the `master`
branch.

A note about the `master` branch:

- Anything in the master branch is deployable, builds successfully and
  is tested to work. The CI/CD system performs both integration and unit
  tests, but should be considered as only a filter to immediately
  highlight PRs which would break the master branch and therefore need
  to be either discarded or amended. Automated checks are no substitute
  for code review, so all PRs are manually reviewed prior to merge.

- Direct commits to the master branch are **prohibited**, with the
  only exception being a core-dev pushing a signed git-tag signifying a
  release.

[github_flow]: https://guides.github.com/introduction/flow/

### Fork, clone and setup upstream remote

The following instructions outline the recommended procedure for
creating a fork of this repository in order to contribute changes.

Firstly, click the `fork` button at the top of the page. Once forked,
clone your fork and set an upstream remote to keep track of changes.

```shell
git clone git@github.com:USERNAME/relay.git

cd relay
git remote add upstream git@github.com:wireleap/relay.git
git checkout master
git pull --tags upstream master
git config commit.gpgsign true
```

### Create a feature branch and make your changes

Create a descriptively named topic branch based on the `master` branch.
Please take care to only address **one** issue/bug/feature per pull
request.

```shell
git checkout master
git pull --tags upstream master
git checkout -b DESCRIPTIVE_BRANCH_NAME
```

When making your changes, test and commit as you go. Try to make commits
that capture an atomic change to the codebase. Source code should be
documented where necessary and the rationale for changes included in
commits should be clear.

If a commit resolves a known issue or relates to other commits or PRs,
please refer to them.

### Unit testing

The unit tests can either be run on your host or within docker using the
official golang docker image.

```shell
# run unit tests on host
./contrib/run-tests.sh

# run unit tests in docker
./contrib/docker/run-tests.sh

# run unit tests in docker (specify cache for faster subsequent tests)
mkdir -p build/.deps
DEPS_CACHE=build/.deps ./contrib/docker/run-tests.sh
```

### Rebase on master if needed

It can happen that as you were working on a feature, the state of the
`upstream/master` branch has changed due to merging other pull requests.
In this case, rebase your topic branch on top of the `master` branch. If
needed, resolve merge conflicts.

```shell
git checkout master
git fetch upstream
git merge upstream/master
git rebase --interactive master DESCRIPTIVE_BRANCH_NAME
```

After every change to the git history of your topic branch, perform
testing to avoid regressions.

### Push changes and submit a pull request

When you think the topic branch is ready for merging, passes all tests,
all changes are committed with appropriate commit messages, and your
topic branch is based on the current state of the `upstream/master`
branch, push them to the **topic branch** (not master) of your fork.

```shell
# push changes
git push origin DESCRIPTIVE_BRANCH_NAME

# if you have already pushed commits to a topic branch, and later
# performed a rebase on top of master, a force push will be required
git push --force origin DESCRIPTIVE_BRANCH_NAME
```

Once pushed, follow the link specified in the `git push` output. Give
your changes a last-minute correctness check, and supply the high-level
description of the changes.

Finally, click `create pull request` so the reviewers can review and
approve the changes, or request modifications prior to performing the
merge.

### Review process and merge

The pull request may be approved or additional modifications might be
requested by one of the reviewers. If modifications are requested,
commit and push more changes to the **same** topic branch and they will
be included in the original pull request until it is ultimately closed.

Branch protection rules are in place. They include:

- Requiring all commits in PRs to be signed.
- Requiring all integration and unit tests to complete successfully.
- Requiring at least one approval from a core-dev.

If there is an issue with the proposed changes, modifications should be
requested. For discussions on the rationale of certain choices in the
code, GitHub comments in the respective files can be left for the author
of the pull request to address.

Please note that every merged pull request is considered final and it is
always better to hold off on merging a pull request than have to open
another one correcting the changes from the first one. Additionally, it
is also sometimes a good idea to create pull requests towards another
PRs topic branch instead of master. This allows unifying multiple sets
of changes from different developers within the scope of a single PR.

Merging changes that are not unanimously approved by all reviewers is
**not** allowed unless special arrangements are in place (e.g. a
reviewer is away and explicitly asked to not wait on them for merging
changes).

Once the above is satisfied and all the reviewers have approved the
changes, the last person who gives their approval and has merge
permissions will close the pull request by merging it into the `master`
branch. However, if the author of the pull request has merge
permissions, they may perform the merge subject to the above.

## License

The MIT License (MIT)
