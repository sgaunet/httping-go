package httping_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sgaunet/httping-go/pkg/httping"
)

const testTimeout = 2 * time.Second

func newPinger(t *testing.T, url string, sleep, timeout time.Duration) *httping.Pinger {
	t.Helper()

	p, err := httping.New(url, sleep, timeout)
	if err != nil {
		t.Fatalf("New(%q) returned unexpected error: %v", url, err)
	}

	return p
}

func TestNewRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	p, err := httping.New("", httping.DefaultSleep, httping.DefaultTimeout)
	if !errors.Is(err, httping.ErrEmptyURL) {
		t.Fatalf("New(\"\") error = %v, want ErrEmptyURL", err)
	}

	if p != nil {
		t.Errorf("New(\"\") returned pinger %v, want nil", p)
	}
}

func TestCheckSuccess(t *testing.T) {
	t.Parallel()

	body := "hello httping"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "httping" {
			t.Errorf("User-Agent = %q, want %q", got, "httping")
		}

		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}

		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	res, err := newPinger(t, srv.URL, httping.DefaultSleep, testTimeout).Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	if res.Size != len(body) {
		t.Errorf("Size = %d, want %d", res.Size, len(body))
	}

	if res.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", res.Duration)
	}
}

func TestCheckReportsStatusCodes(t *testing.T) {
	t.Parallel()

	for _, want := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", want), func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(want)
			}))
			defer srv.Close()

			res, err := newPinger(t, srv.URL, httping.DefaultSleep, testTimeout).Check(context.Background())
			if err != nil {
				t.Fatalf("Check() returned unexpected error: %v", err)
			}

			if res.StatusCode != want {
				t.Errorf("StatusCode = %d, want %d", res.StatusCode, want)
			}
		})
	}
}

func TestCheckConnectionError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening any more

	_, err := newPinger(t, url, httping.DefaultSleep, testTimeout).Check(context.Background())
	if err == nil {
		t.Fatal("Check() against a closed server returned nil error, want error")
	}

	if !strings.Contains(err.Error(), "cannot connect to") {
		t.Errorf("error = %q, want it to mention \"cannot connect to\"", err)
	}
}

func TestCheckInvalidURL(t *testing.T) {
	t.Parallel()

	_, err := newPinger(t, "http://\x7f/", httping.DefaultSleep, testTimeout).Check(context.Background())
	if err == nil {
		t.Fatal("Check() with an invalid URL returned nil error, want error")
	}

	if !strings.Contains(err.Error(), "cannot create request for") {
		t.Errorf("error = %q, want it to mention \"cannot create request for\"", err)
	}
}

// TestCheckHonoursTimeout covers issue #9: a slow endpoint must fail fast
// rather than hang the monitoring loop.
func TestCheckHonoursTimeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))

	defer func() {
		close(release)
		srv.Close()
	}()

	start := time.Now()

	_, err := newPinger(t, srv.URL, httping.DefaultSleep, 50*time.Millisecond).Check(context.Background())
	if err == nil {
		t.Fatal("Check() against a hanging server returned nil error, want timeout error")
	}

	if elapsed := time.Since(start); elapsed > testTimeout {
		t.Errorf("Check() took %v, want it to give up near the 50ms timeout", elapsed)
	}
}

// TestCheckReusesConnection covers issue #8: one http.Client is shared across
// probes, so a second probe must ride the pooled keep-alive connection instead
// of opening a new one.
func TestCheckReusesConnection(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		newConns int
	)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			newConns++
			mu.Unlock()
		}
	}
	srv.Start()

	defer srv.Close()

	pinger := newPinger(t, srv.URL, httping.DefaultSleep, testTimeout)

	const probes = 3
	for i := range probes {
		if _, err := pinger.Check(context.Background()); err != nil {
			t.Fatalf("Check() probe %d returned unexpected error: %v", i+1, err)
		}
	}

	mu.Lock()
	got := newConns
	mu.Unlock()

	if got != 1 {
		t.Errorf("server saw %d new connections across %d probes, want 1 (connection not reused)", got, probes)
	}
}

func TestCloseAllowsFreshConnectionAfterwards(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	pinger := newPinger(t, srv.URL, httping.DefaultSleep, testTimeout)

	if _, err := pinger.Check(context.Background()); err != nil {
		t.Fatalf("Check() before Close returned unexpected error: %v", err)
	}

	pinger.Close()

	// Close only drops pooled connections; it must not break the client.
	if _, err := pinger.Check(context.Background()); err != nil {
		t.Fatalf("Check() after Close returned unexpected error: %v", err)
	}
}

func TestRunWritesOneLinePerProbe(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "body")
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()

	var out strings.Builder
	if err := newPinger(t, srv.URL, 20*time.Millisecond, testTimeout).Run(ctx, &out); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("Run() wrote %d line(s), want at least 2:\n%s", len(lines), out.String())
	}

	if !strings.Contains(lines[0], "connected to "+srv.URL) {
		t.Errorf("first line = %q, want it to mention the target URL", lines[0])
	}

	if !strings.Contains(lines[0], "seq=1") || !strings.Contains(lines[1], "seq=2") {
		t.Errorf("sequence numbers not incrementing:\n%s", out.String())
	}
}

func TestRunReportsErrorsAndKeepsGoing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()

	var out strings.Builder
	if err := newPinger(t, url, 20*time.Millisecond, testTimeout).Run(ctx, &out); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "ERROR: seq=1") {
		t.Errorf("output = %q, want it to contain \"ERROR: seq=1\"", out.String())
	}

	if !strings.Contains(out.String(), "ERROR: seq=2") {
		t.Errorf("Run() stopped after the first failure; want it to keep probing:\n%s", out.String())
	}
}

func TestRunStopsWhenContextCancelled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: exactly one probe, then return

	var out strings.Builder
	if err := newPinger(t, srv.URL, time.Hour, testTimeout).Run(ctx, &out); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if got := strings.Count(out.String(), "\n"); got != 1 {
		t.Errorf("Run() wrote %d lines, want 1 before honouring cancellation:\n%s", got, out.String())
	}
}

func TestRunPropagatesWriteError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	err := newPinger(t, srv.URL, time.Millisecond, testTimeout).Run(context.Background(), failingWriter{})
	if err == nil {
		t.Fatal("Run() with a failing writer returned nil error, want error")
	}

	if !strings.Contains(err.Error(), "cannot write probe output") {
		t.Errorf("error = %q, want it to mention \"cannot write probe output\"", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

var errWriteFailed = errors.New("write failed")

func TestFormatResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		seq  int
		res  httping.Result
		err  error
		want string
	}{
		{
			name: "success",
			seq:  1,
			res: httping.Result{
				Duration:   1500 * time.Microsecond,
				Size:       42,
				StatusCode: http.StatusOK,
			},
			want: "connected to http://example.com, seq=1 time=1.500 bytes=42 StatusCode=200\n",
		},
		{
			name: "rounds to three decimals",
			seq:  7,
			res: httping.Result{
				Duration:   761159 * time.Microsecond,
				Size:       206779,
				StatusCode: http.StatusNotFound,
			},
			want: "connected to http://example.com, seq=7 time=761.159 bytes=206779 StatusCode=404\n",
		},
		{
			name: "error line ignores result",
			seq:  3,
			res:  httping.Result{Duration: time.Second, Size: 9, StatusCode: http.StatusOK},
			err:  errWriteFailed,
			want: "ERROR: seq=3 write failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := httping.FormatResult("http://example.com", tt.seq, tt.res, tt.err); got != tt.want {
				t.Errorf("FormatResult() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
