# Wireleap client API

## Table of Contents

- [Introduction](#introduction)
- [Errors](#errors)
- [API](#api)
    - [Relay](#status)
        - [The relay object](#the-relay-object)
        - [Get relay status](#get-relay-status)
## Introduction

> Port location

```shell
$ cat "./config.json" | jq ".rest_api.address"
```

The Wireleap relay provides an REST API endpoint on unix socket and TCP port.
Requests and responses are JSON-encoded, if not specified otherwhise,
and uses standard HTTP response codes.

## Errors

> HTTP status code summary

```
200 - OK                 Everything worked as expected
400 - Bad Request        The request was unacceptable
402 - Request Failed     Parameters valid but the request failed
403 - Forbidden          No permission to perform request
404 - Not Found          The requested resource doesn't exist
405 - Method Not Allowed The requested resource exists but method not supported
500 - Error              Internal server error
501 - Error              Not implemented
```

The REST API uses conventional HTTP response codes to indicate the
success or failure of an API request. In general: Codes in the `2xx` range
indicate success. Codes in the `4xx` range indicate an error that failed
given the information provided (e.g., a required parameter was omitted).
Codes in the `5xx` range indicate an error with the REST API server.

Most errors may include the `code`, `desc` and `cause` of the error in
the body of the response.

# API

## Status

> Endpoints

```
GET  /api/status
```

The Wireleap relay returns basic opertation metric.

### The relay object

> The relay object

```json
{
  "controller_started": true,
  "network_usage": {
    "timeframe_since": 1661811346792,
    "timeframe_until": 1664403346792,
    "cap": 21990232555520,
    "usage": 0
  },
  "relay_status": [
    {
      "id": "LWC14711LBBJ3qmlfomYm0HrbDZd4aD8bQhP_haj9x0",
      "address": "wireleap://spyder-ub-1.stuker.es:443",
      "role": "fronting",
      "status": {
        "enrolled": true,
        "network_cap_reached": false
      },
      "network_cap": 21990232555520,
      "network_usage": 0
    }
  ]
}
```

#### Attributes

Key                                        | Type     | Comment
---                                        | ----     | -------
controller_started                         | `bool`   | Controller status
network_usage.timeframe_since              | `int64`  | Current period start (epoch millis)
network_usage.timeframe_until              | `int64`  | Current period end (epoch millis)
network_usage.cap                          | `int64`  | Global network cap (bytes)
network_usage.usage                        | `int64`  | Global network usage (bytes)
relay_status[X].id                         | `string` | Contract public key
relay_status[X].address                    | `string` | Address of relay
relay_status[X].role                       | `string` | Type of relay (`fronting`, `backing`, `entropic`)
relay_status[X].status.enrolled            | `bool`   | Is relay enrolled
relay_status[X].status.network_cap_reached | `bool`   | Has relay reached contract of global cap
relay_status[X].network_cap                | `int64`  | Contract network cap (bytes)
relay_status[X].network_usage              | `int64`  | Contract network usage (bytes)

### Get controller status

> Get controller status

```shell
$ curl $URL/api/status
```

Retrieves the current status of the controller.

#### Parameters

None

#### Returns

The `relay` object.