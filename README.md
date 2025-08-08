[![GitHub release](https://img.shields.io/github/release/sgaunet/httping-go.svg)](https://github.com/sgaunet/httping-go/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/httping-go)](https://goreportcard.com/report/github.com/sgaunet/httping-go)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/httping-go/total)
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
        time to sleep between two tries. (default: 200) (default 200)
  -u string
        url to "ping"
exit status 2
```


# build

```
go build . 
```

# task

Use task to generate the binary and execute it when source code changes.

```
task -w run
```
