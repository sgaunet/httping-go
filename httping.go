package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func check(url string) (urltime float64, urlsize int, status int) {
	t0 := time.Now()
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Cannot connect to %s", url)
	} else {
		req.Proto = "HTTP/1.1"
		req.ProtoMinor = 0
		req.Header.Set("User-Agent", "httping")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Cannot connect to %s\n", url)
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			url_size := len(body)
			msec := time.Since(t0)
			url_time := msec.Seconds() * float64(time.Second/time.Millisecond)
			statusCode := resp.StatusCode
			return url_time, url_size, statusCode
		}
	}
	return
}

var version string = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var url string
	var sleepMs int
	var vOption bool
	flag.StringVar(&url, "u", "", "url to \"ping\"")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.IntVar(&sleepMs, "s", 200, "time to sleep between two tries. (default: 200)")
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
		seq = seq + 1
		timeOfRequest, contentLength, statusCode := check(url)
		fmt.Printf("connected to %s, seq=%d time=%s bytes=%d StatusCode=%d\n", url, seq, strconv.FormatFloat(timeOfRequest, 'f', 3, 64), contentLength, statusCode)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}
