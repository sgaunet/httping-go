[![GitHub release](https://img.shields.io/github/release/sgaunet/httping-go.svg)](https://github.com/sgaunet/httping-go/releases/latest)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/httping-go/total)
[![Coverage](https://raw.githubusercontent.com/wiki/sgaunet/httping-go/coverage-badge.svg)](https://github.com/sgaunet/httping-go/actions/workflows/coverage.yml)
[![Snapshot Build](https://github.com/sgaunet/httping-go/actions/workflows/snapshot.yml/badge.svg)](https://github.com/sgaunet/httping-go/actions/workflows/snapshot.yml)
[![Release Build](https://github.com/sgaunet/httping-go/actions/workflows/release.yml/badge.svg)](https://github.com/sgaunet/httping-go/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/sgaunet/httping-go.svg)](LICENSE)

# httping

httping is a small program to request http server in order to print statuscode and the time to get the response.

Example :

```
$ httping -u https://www.github.com -s 500
connected to https://www.github.com, seq=1 time=761.159 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=2 time=147.326 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=3 time=143.971 bytes=206779 StatusCode=200
connected to https://www.github.com, seq=4 time=138.060 bytes=206779 StatusCode=200
^Csignal: interrupt
```

# Usage

```
httping:
  -s int
        time to sleep between two tries, in milliseconds (default 200)
  -t int
        timeout of an HTTP request, in seconds (default 10)
  -u string
        url to "ping"
  -v    Get version
```

A single HTTP client is reused across probes, so connections are pooled with
keep-alive instead of being re-established for every request. Each request is
bounded by `-t`, so an unresponsive endpoint is reported as an error and
monitoring carries on instead of hanging.

```
$ httping -u https://www.github.com -s 500 -t 5
```

Press `Ctrl-C` to stop.

# install

```
go install github.com/sgaunet/httping-go/cmd/httping-go@latest
```

# build

```
go build ./cmd/httping-go
```

# task

Use task to generate the binary and execute it when source code changes.

```
task -w run
```
