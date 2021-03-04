package main

import (
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

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage : httping <url>")
		os.Exit(1)
	}

	url := os.Args[1]
	seq := 0
	for {
		seq = seq + 1
		timeOfRequest, contentLength, statusCode := check(url)
		fmt.Printf("connected to %s, seq=%d time=%s bytes=%d StatusCode=%d\n", url, seq, strconv.FormatFloat(timeOfRequest, 'f', 3, 64), contentLength, statusCode)
		time.Sleep(200 * time.Millisecond)
	}
}
