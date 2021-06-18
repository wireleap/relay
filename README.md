# wireleap relay

This is the [wireleap](https://wireleap.com) relay. The binary name is
`wireleap-relay`.

> Simplified traffic flow diagram

```shell
        +--------+                          +-------+
 -----> |fronting| -----------------------> |backing| ---->
        +--------+                          +-------+

        +--------+        +--------+        +-------+
 -----> |fronting| -----> |entropic| -----> |backing| ---->
        +--------+        +--------+        +-------+
```

A relay provides relaying services for a contract. Each relay enrolled
into a contract assigns itself a `role` related to its position in
the connection circuit.

Role | Comment
---- | -------
fronting | provide an on-ramp to the routing layer
entropic | provide additional *optional entropy* to the circuit
backing | provide an exit from the routing layer

## Relay configuration


> Example1: Relay configuration (config.json)

```json
{
    "address": "127.0.0.1:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://relay1.example.com:443/wireleap",
            "role": "fronting"
        }
    }
}
```

> Example2: Relay configuration (config.json)

```json
{
    "address": "0.0.0.0:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "backing",
            "key": "backing:secretkey"
        },
        "https://contract2.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "entropic"
        }
    }
}
```

> Example3: Relay configuration (config.json)

```json
{
    "address": "0.0.0.0:13490",
    "archive_dir": "archive/sharetokens",
    "auto_submit_interval": "5m0s",
    "contracts": {
        "https://contract1.example.com": {
            "address": "wireleap://relay1.example.com:13490",
            "role": "backing",
            "key": "backing:secretkey"
        },
        "https://contract2.example.com": {
            "address": "wireleap://relay1.example.com:443/wireleap",
            "role": "fronting"
        }
    }
}
```

Relays are configured through the file `config.json`, which lists
contracts the relay supports along with a specific configuration per
each contract as well as the daemon configuration.

Attribute | Type | Comment
--------- | ---- | -------
address | `string` | address to bind to (`host:port`)
archive_dir | `string` | path to archive submitted sharetokens (optional)
auto_submit_interval | `string` | interval between sharetoken submission retries
contracts.X | `string` | service contract endpoint url
contracts.X.address | `string` | `wireleap://host:port`
contracts.X.role | `string` | `fronting` `entropic` `backing`
contracts.X.key | `string` | `user:password` format enrollment key if required
contracts.X.update_channel | `string` | update channel (default: `"default"`)
auto_upgrade | `bool` | automatically upgrade this relay (default: `true`)

Note: the default `ulimit -n` value of `1024` on most machines would
likely prove too low for running a production relay. Consider changing
it to a higher value, e. g. `sudo ulimit -n 65535`.

### Protocol encapsulation

A HTTP/2 connection is used as transport and the individual connections
are streams inside it.  The HTTP/2 connection is opaque and valid
traffic for the webserver to proxy. Once the proxying webserver has
performed the TLS and H/2 negotiation (settings frame, etc.) and
bidirectional streaming data transfer is setup, the encapsulated traffic
is seamlessly proxied to and from the relay daemon `address`.

## Relay daemon

> Setup

```shell
cd $HOME/wireleap-relay

# initialize relay directory
./wireleap-relay init

# create the relay configuration
$EDITOR config.json

# start the daemon in the foreground
./wireleap-relay start --fg

# tip: reload config file
./wireleap-relay reload
```

[Install](#installation) or [build](#building) `wireleap-relay`,
generate certificates and a keypair specific for the relay, create a
configuration and start the daemon.

Note: the directory where `wireleap-relay` looks for its files such as
`config.json` is the enclosing directory of the `wireleap-relay` binary.

## Relay daemon supervisor

> Example systemd unit file (wireleap-relay.service)

```systemd
[Unit]
Description=Wireleap relay process
After=multi-user.target
StartLimitIntervalSec=200
StartLimitBurst=5

[Service]
Type=forking
User=relay
Group=relay
RootDirectory=/path/to/wireleap-relay
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

To keep the relay process up and running at all times, the use of a
process supervisor like [systemd][systemd] is recommended. The following
is a suitable systemd unit file template:

To use it in your deployment, you first need to create the new relay
user via `useradd -rMU relay`. Then, please replace
`/path/to/wireleap-relay` with the real path to `wireleap-relay` and
install/enable the unit file via `systemctl enable`. Ensure that the
permissions on the directory enclosing the `wireleap-relay` executable
allow the `relay` user to access the binary. Check the service status
with `systemctl status wireleap-relay.service`.

This will watch the relay daemon process and restart it if it fails for
any reason.

[systemd]: https://en.wikipedia.org/wiki/Systemd

# Relay web server proxying

> Example Apache configuration

```apache
<IfModule ssl_module>
 <IfModule http2_module>
   Listen 443 https
   Protocols h2
   <VirtualHost *:443>
     ServerAdmin admin@relay1.example.com
     SSLCertificateFile /path/to/relay.crt
     SSLCertificateKeyFile /path/to/relay.key
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

> Example Nginx configuration

```nginx
server {
    listen 443 ssl http2;
    server_name relay1.example.com;
    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;
    ssl_protocols TLSv1.3;

    location /wireleap {
        grpc_pass grpcs://127.0.0.1:13490;
        grpc_ssl_protocols TLSv1.3;
    }
}
```

Note: the examples in this section pertain to the fronting relay
configuration.

When the fronting relay daemon is behind a proxying web server that
supports HTTP/2, such as Apache or Nginx, the web server can be
configured to proxy connections to the daemon while simultaneously
serving the regular website traffic.

The relevant config sections of note are `PROTOCOL/LISTEN` and
`LOCATION`.

# Relay settlement

A service contract defines the service parameters and facilitates
disbursing funds provided by a customer to service providers
in proportion to service provided based on proof of service.

The proof of service in question are the sharetokens accumulated by a
relay when relaying user traffic.

## Submitting sharetokens

Sharetokens are accumulated when relaying user traffic and
automatically submitted to the contract upon servicekey expiration. If
for some reason the submission fails, it will be retried based on the
`auto_submit_interval` as defined in the relay configuration.

Note: Sharetokens need to be submitted to the contract during the
submission window (from servicekey expiration plus
``settlement.submission_window``) as defined by the service contract.

## Checking status

> Query balance

```shell
# show available balance, sharetokens awaiting settlement window,
# and last withdrawal
./wireleap-relay balance
```

Once sharetokens are submitted for settlement, their status can be
queried with the `balance` command. When the settlement window closes
and all verification checks are complete, the final calculation is
performed. Based on the calculation, the relay's balance will be
credited, also shown per the `balance` command.

Note: the available balance shown is the integer part of the real,
internally stored balance of the relay. This ensures that settlement
always results in a fair assignment of relay shares. It may also lead
to sub-one cent balance increases under certain conditions which will
not affect the available balance immediately but increase the
internally stored balance.

## Initiating a withdrawal request

> Initiate withdrawal request

```shell
# show available balance
./wireleap-relay balance \
    --contract https://contract1.example.com

# request withdrawal
./wireleap-relay withdraw \
    --contract https://contract1.example.com \
    --destination acct_1032D82eZvKYlo2C \
    --amount 150
```

A relay operator may issue a withdrawal request up to or equal to their
available balance.

The contract operator defines the `payout` configuration in the [service
contract configuration](#service-contract--contract-configuration).
Information regarding the `--destination` can be obtained by visiting
the link as defined in `payout.info`.

`curl -s https://contract1.example.com/info | jq -r '.payout.info'`

# Relay upgrades

The [precompiled binary](#installation) of `wireleap-relay` includes
both automatic and manual upgrade functionality. Due to the protocol
[versioning](#versioning), it is highly recommended to keep relays up to
date.

## Automatic upgrades

> Automatic upgrade configuration

```json
{
    "auto_upgrade": true,
    "contracts": {
        "https://contract1.example.com": {
            "update_channel": "default"
        }
    }
}
```

When `auto_upgrade` is set to `true` or not present in the relay's
`config.json`, the relay will attempt automatic upgrades whenever it
receives an update notification from the directory on the update channel
specified in the enrollment config (`contracts.X`). If an upgrade fails,
a best-effort rollback is performed and the affected version is skipped.

The update channels supported by the directory and the respective latest
versions are exposed via the directories `/info` endpoint.


## Manual upgrades

> Manual upgrade

```shell
cd $HOME/wireleap-relay

# perform interactive upgrade to latest release
./wireleap-relay upgrade

# rollback if needed
./wireleap-relay rollback
```

The upgrade process is interactive so you will have the possibility
to accept or decline based on the changelog for the new release version.

If the upgrade was successful, the old binary is not deleted but kept as
`wireleap-relay.prev` (for rollback purposes in case issues manifest
post-upgrade).

