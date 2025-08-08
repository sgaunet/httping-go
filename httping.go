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

func check(url string) (float64, int, int) {
	t0 := time.Now()
	client := &http.Client{}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("Cannot connect to %s", url)
	}
	
	req.Proto = "HTTP/1.1"
	req.ProtoMinor = 0
	req.Header.Set("User-Agent", "httping")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Cannot connect to %s\n", url)
	}
	
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()
	
	body, _ := io.ReadAll(resp.Body)
	urlSize := len(body)
	msec := time.Since(t0)
	urlTime := msec.Seconds() * float64(time.Second/time.Millisecond)
	statusCode := resp.StatusCode
	return urlTime, urlSize, statusCode
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
		timeOfRequest, contentLength, statusCode := check(url)
		fmt.Printf("connected to %s, seq=%d time=%s bytes=%d StatusCode=%d\n",
			url, seq, strconv.FormatFloat(timeOfRequest, 'f', 3, 64),
			contentLength, statusCode)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}