# Wireleap relay command line reference

## Table of contents

- [wireleap-relay](#wireleap-relay)
- [wireleap-relay init](#wireleap-relay-init)
- [wireleap-relay start](#wireleap-relay-start)
- [wireleap-relay stop](#wireleap-relay-stop)
- [wireleap-relay restart](#wireleap-relay-restart)
- [wireleap-relay reload](#wireleap-relay-reload)
- [wireleap-relay status](#wireleap-relay-status)
- [wireleap-relay upgrade](#wireleap-relay-upgrade)
- [wireleap-relay rollback](#wireleap-relay-rollback)
- [wireleap-relay check-config](#wireleap-relay-check-config)
- [wireleap-relay balance](#wireleap-relay-balance)
- [wireleap-relay withdraw](#wireleap-relay-withdraw)
- [wireleap-relay version](#wireleap-relay-version)

## wireleap-relay 

```
$ wireleap-relay help
Usage: wireleap-relay COMMAND [OPTIONS]

Commands:
  help            Display this help message or help on a command
  init            Generate ed25519 keypair and TLS cert/key
  start           Start wireleap-relay daemon
  stop            Stop wireleap-relay daemon
  restart         Restart wireleap-relay daemon
  reload          Reload wireleap-relay daemon configuration
  status          Report wireleap-relay daemon status
  upgrade         Upgrade wireleap-relay to the latest version per directory
  rollback        Undo a partially completed upgrade
  check-config    Validate wireleap-relay config file
  balance         Show balance, pending sharetokens and last withdrawal
  withdraw        Withdraw available funds from balance
  version         Show version and exit

Run 'wireleap-relay help COMMAND' for more information on a command.
```

## wireleap-relay init

```
$ wireleap-relay help init
Usage: wireleap-relay init

Generate ed25519 keypair and TLS cert/key
```

## wireleap-relay start

```
$ wireleap-relay help start
Usage: wireleap-relay start [options]

Start wireleap-relay daemon

Options:
  --fg  Run in foreground, don't detach
```

## wireleap-relay stop

```
$ wireleap-relay help stop
Usage: wireleap-relay stop

Stop wireleap-relay daemon
```

## wireleap-relay restart

```
$ wireleap-relay help restart
Usage: wireleap-relay restart

Restart wireleap-relay daemon
```

## wireleap-relay reload

```
$ wireleap-relay help reload
Usage: wireleap-relay reload

Reload wireleap-relay daemon configuration
```

## wireleap-relay status

```
$ wireleap-relay help status
Usage: wireleap-relay status

Report wireleap-relay daemon status

Exit codes:
  0     wireleap-relay is running
  1     wireleap-relay is not running
  2     could not tell if wireleap-relay is running or not
```

## wireleap-relay upgrade

```
$ wireleap-relay help upgrade
Usage: wireleap-relay upgrade

Upgrade wireleap-relay to the latest version per directory
```

## wireleap-relay rollback

```
$ wireleap-relay help rollback
Usage: wireleap-relay rollback

Undo a partially completed upgrade
```

## wireleap-relay check-config

```
$ wireleap-relay help check-config
Usage: wireleap-relay check-config

Validate wireleap-relay config file
```

## wireleap-relay balance

```
$ wireleap-relay help balance
Usage: wireleap-relay balance [options]

Show balance, pending sharetokens and last withdrawal

Options:
  --contract string  Service contract URL
```

## wireleap-relay withdraw

```
$ wireleap-relay help withdraw
Usage: wireleap-relay withdraw [options]

Withdraw available funds from balance

Options:
  --amount string       Withdraw given amount from the balance
  --contract string     Service contract URL
  --destination string  Withdraw to this destination
```

## wireleap-relay version

```
$ wireleap-relay help version
Usage: wireleap-relay version [options]

Show version and exit

Options:
  -v   show verbose output
```

