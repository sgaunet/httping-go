// Package main implements the httping-go command line interface.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/sgaunet/httping-go/pkg/httping"
)

// Process exit statuses.
const (
	exitSuccess = 0
	exitFailure = 1
)

// version is injected at build time via ldflags.
var version = "development"

func main() {
	os.Exit(run())
}

// run parses the command line, probes the requested URL until interrupted and
// reports the process exit status. It exists so that deferred cleanup runs
// before main calls os.Exit.
func run() int {
	var (
		url        string
		sleepMs    int
		timeoutSec int
		vOption    bool
	)

	flag.StringVar(&url, "u", "", "url to \"ping\"")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.IntVar(&sleepMs, "s", int(httping.DefaultSleep/time.Millisecond),
		"time to sleep between two tries, in milliseconds")
	flag.IntVar(&timeoutSec, "t", int(httping.DefaultTimeout/time.Second),
		"timeout of an HTTP request, in seconds")
	flag.Parse()

	if vOption {
		fmt.Println(version)

		return exitSuccess
	}

	pinger, err := httping.New(url,
		time.Duration(sleepMs)*time.Millisecond,
		time.Duration(timeoutSec)*time.Second)
	if err != nil {
		flag.PrintDefaults()

		return exitFailure
	}

	defer pinger.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if runErr := pinger.Run(ctx, os.Stdout); runErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", runErr)

		return exitFailure
	}

	return exitSuccess
}
