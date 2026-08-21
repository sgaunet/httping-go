// Package httping probes HTTP endpoints and reports response time, size and status code.
package httping

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Default probe settings.
const (
	// DefaultSleep is the pause between two consecutive probes.
	DefaultSleep = 200 * time.Millisecond
	// DefaultTimeout bounds a single HTTP exchange, body read included.
	DefaultTimeout = 10 * time.Second
)

// timeDecimals is the number of decimals used when printing elapsed milliseconds.
const timeDecimals = 3

// maxIdleConnsPerHost caps the keep-alive connections pooled for the probed
// host. Probes are sequential, so a single pooled connection is enough.
const maxIdleConnsPerHost = 1

// ErrEmptyURL is returned by New when no target URL is provided.
var ErrEmptyURL = errors.New("url must not be empty")

// Result reports the outcome of a single successful probe.
type Result struct {
	// Duration is the time elapsed between sending the request and finishing
	// the body read.
	Duration time.Duration
	// Size is the response body length in bytes.
	Size int
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
}

// Pinger repeatedly probes a single URL. It holds one http.Client that is
// reused for every probe, so connections are pooled between probes instead of
// being re-established each time.
type Pinger struct {
	client    *http.Client
	transport *http.Transport
	url       string
	sleep     time.Duration
}

// New returns a Pinger targeting url, pausing sleep between probes and giving
// up on any single exchange after timeout. It returns ErrEmptyURL if url is
// empty.
func New(url string, sleep, timeout time.Duration) (*Pinger, error) {
	if url == "" {
		return nil, ErrEmptyURL
	}

	transport := newTransport()

	return &Pinger{
		client:    &http.Client{Timeout: timeout, Transport: transport},
		transport: transport,
		url:       url,
		sleep:     sleep,
	}, nil
}

// newTransport returns a transport dedicated to one Pinger. Cloning the default
// transport keeps its proven dial and handshake timeouts while giving the Pinger
// its own connection pool, so probes are not affected by unrelated code closing
// idle connections on the shared default transport.
func newTransport() *http.Transport {
	def, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{MaxIdleConnsPerHost: maxIdleConnsPerHost}
	}

	transport := def.Clone()
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost

	return transport
}

// Close releases the keep-alive connections pooled by the Pinger. The Pinger
// must not be used afterwards.
func (p *Pinger) Close() {
	p.transport.CloseIdleConnections()
}

// Check performs a single HTTP GET and reports the elapsed time, the response
// size in bytes and the HTTP status code. Transient failures are returned as
// errors so callers can keep monitoring instead of terminating.
func (p *Pinger) Check(ctx context.Context) (Result, error) {
	t0 := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return Result{}, fmt.Errorf("cannot create request for %s: %w", p.url, err)
	}

	req.Proto = "HTTP/1.1"
	req.ProtoMinor = 0
	req.Header.Set("User-Agent", "httping")

	resp, err := p.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("cannot connect to %s: %w", p.url, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("cannot read response body from %s: %w", p.url, err)
	}

	return Result{
		Duration:   time.Since(t0),
		Size:       len(body),
		StatusCode: resp.StatusCode,
	}, nil
}

// Run probes the URL until ctx is cancelled, writing one line per probe to w.
// A failed probe is reported and the loop continues, so monitoring survives
// transient errors. Run returns nil once ctx is cancelled.
func (p *Pinger) Run(ctx context.Context, w io.Writer) error {
	for seq := 1; ; seq++ {
		res, err := p.Check(ctx)

		if _, writeErr := io.WriteString(w, FormatResult(p.url, seq, res, err)); writeErr != nil {
			return fmt.Errorf("cannot write probe output: %w", writeErr)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(p.sleep):
		}
	}
}

// FormatResult renders a single probe as the line written by Run. A non-nil
// err produces an error line and res is ignored.
func FormatResult(url string, seq int, res Result, err error) string {
	if err != nil {
		return fmt.Sprintf("ERROR: seq=%d %v\n", seq, err)
	}

	msec := float64(res.Duration) / float64(time.Millisecond)

	return fmt.Sprintf("connected to %s, seq=%d time=%s bytes=%d StatusCode=%d\n",
		url, seq, strconv.FormatFloat(msec, 'f', timeDecimals, 64),
		res.Size, res.StatusCode)
}
