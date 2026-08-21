// Package main implements a simple HTTP ping utility that continuously tests HTTP endpoints.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const defaultSleepMs = 200

// check performs a single HTTP GET request against url and reports the elapsed
// time in milliseconds, the response size in bytes and the HTTP status code.
// Transient failures are returned as errors so callers can keep monitoring
// instead of terminating.
func check(url string) (float64, int, int, error) {
	t0 := time.Now()
	client := &http.Client{}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("cannot create request for %s: %w", url, err)
	}

	req.Proto = "HTTP/1.1"
	req.ProtoMinor = 0
	req.Header.Set("User-Agent", "httping")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("cannot connect to %s: %w", url, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("cannot read response body from %s: %w", url, err)
	}

	urlSize := len(body)
	msec := time.Since(t0)
	urlTime := msec.Seconds() * float64(time.Second/time.Millisecond)
	statusCode := resp.StatusCode

	return urlTime, urlSize, statusCode, nil
}

var version = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var url string
	var sleepMs int
	var vOption bool
	flag.StringVar(&url, "u", "", "url to \"ping\"")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.IntVar(&sleepMs, "s", defaultSleepMs, "time to sleep between two tries. (default: 200)")
	flag.Parse()

	if vOption {
		printVersion()
		os.Exit(0)
	}
	if len(url) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	seq := 0
	for {
		seq++
		timeOfRequest, contentLength, statusCode, err := check(url)
		if err != nil {
			fmt.Printf("ERROR: seq=%d %v\n", seq, err)
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			continue
		}
		fmt.Printf("connected to %s, seq=%d time=%s bytes=%d StatusCode=%d\n",
			url, seq, strconv.FormatFloat(timeOfRequest, 'f', 3, 64),
			contentLength, statusCode)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}
