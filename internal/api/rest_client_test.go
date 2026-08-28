package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// newTestClient builds a RestClient pointed at the given base URL with tiny,
// deterministic retry/backoff settings so tests run fast and predictably. By
// default the rate limiter is effectively unlimited; individual tests override
// it when they need to exercise shared pacing.
func newTestClient(t *testing.T, baseURL string) *RestClient {
	t.Helper()
	c, err := NewRestClient(baseURL, "test-token")
	if err != nil {
		t.Fatalf("NewRestClient: %v", err)
	}
	c.baseBackoff = time.Millisecond
	c.maxBackoff = 50 * time.Millisecond
	c.randFloat = func() float64 { return 1.0 }
	c.limiter = rate.NewLimiter(rate.Inf, 1)
	return c
}

// countingHandler returns a handler that serves `first` for the first request
// and `then` for every subsequent request, tracking the number of hits.
func countingHandler(hits *int32, first, then http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(hits, 1)
		if n == 1 {
			first(w, r)
			return
		}
		then(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func TestDoRequest_RetryAfterDeltaSeconds(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(countingHandler(&hits,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusTooManyRequests, `{"message":"rate limited"}`)
		},
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"ok":true}`)
		},
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	body, resp, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 requests (retry once), got %d", got)
	}
}

func TestDoRequest_RetryAfterHTTPDate(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(countingHandler(&hits,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", time.Now().Add(time.Second).UTC().Format(http.TimeFormat))
			writeJSON(w, http.StatusTooManyRequests, `{"message":"rate limited"}`)
		},
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"ok":true}`)
		},
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, resp, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

func TestDoRequest_RateLimitReset(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(countingHandler(&hits,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("RateLimit-Remaining", "0")
			w.Header().Set("RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
			writeJSON(w, http.StatusTooManyRequests, `{"message":"rate limited"}`)
		},
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"ok":true}`)
		},
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, resp, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

func TestDoRequest_5xxThenSuccess(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(countingHandler(&hits,
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusInternalServerError, `{"message":"boom"}`)
		},
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"ok":true}`)
		},
	))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, resp, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
}

func TestDoRequest_CtxCancelledDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always signal a long wait so the client would otherwise sleep for seconds.
		w.Header().Set("Retry-After", "3600")
		writeJSON(w, http.StatusTooManyRequests, `{"message":"rate limited"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	// Allow a large clamp so the backoff would be long if not cancelled.
	c.maxBackoff = 5 * time.Second
	c.baseBackoff = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err := c.doRequest(ctx, "GET", "/test", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("expected prompt return on cancel, took %v", elapsed)
	}
}

func TestDoRequest_RetryBudgetExhausted(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeJSON(w, http.StatusInternalServerError, `{"message":"always failing"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.maxRetries = 2

	_, _, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Fatalf("expected 3 attempts (maxRetries+1), got %d", got)
	}
}

func TestDoRequest_NonRetryableReturnsImmediately(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, _, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err == nil {
		t.Fatal("expected an error for 404, got nil")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 request (no retry on 404), got %d", got)
	}
}

// paginationHandler serves the X-Total probe (per_page=1) and full-pagination
// pages (per_page=100). probeHeader controls the X-Total header on the probe;
// pages maps a page number to the number of items returned for per_page=100.
func paginationHandler(probeHeader string, pages map[string]int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		perPage := q.Get("per_page")
		if perPage == "1" {
			if probeHeader != "" {
				w.Header().Set("X-Total", probeHeader)
			}
			// Probe body content is irrelevant to counting.
			writeJSON(w, http.StatusOK, "[]")
			return
		}
		n := pages[q.Get("page")]
		items := make([]string, n)
		for i := range items {
			items[i] = "{}"
		}
		writeJSON(w, http.StatusOK, "["+join(items, ",")+"]")
	}
}

func join(items []string, sep string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func TestGetCount_MissingXTotalFallsBackToPagination(t *testing.T) {
	// No X-Total on probe -> paginate: page 1 full (100), page 2 partial (50) => 150.
	srv := httptest.NewServer(paginationHandler("", map[string]int{"1": 100, "2": 50}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 150 {
		t.Fatalf("expected 150 from pagination fallback, got %d", count)
	}
}

func TestGetCount_NonNumericXTotalFallsBackToPagination(t *testing.T) {
	// Non-numeric X-Total must be treated as absent, triggering pagination.
	srv := httptest.NewServer(paginationHandler("not-a-number", map[string]int{"1": 10}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 10 {
		t.Fatalf("expected 10 from pagination fallback, got %d", count)
	}
}

func TestGetCount_LegitZeroXTotal(t *testing.T) {
	// A present, numeric X-Total of 0 is a real zero, distinct from a failure.
	srv := httptest.NewServer(paginationHandler("0", map[string]int{}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestGetCount_PresentXTotal(t *testing.T) {
	srv := httptest.NewServer(paginationHandler("42", map[string]int{}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected 42 from X-Total, got %d", count)
	}
}

func TestGetCount_TransportErrorReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, `{"message":"down"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.maxRetries = 1
	_, err := c.getBranchCount(context.Background(), 1)
	if err == nil {
		t.Fatal("expected an error when the count request keeps failing, got nil")
	}
}

func TestSharedLimiter_ConcurrentRetries(t *testing.T) {
	const goroutines = 8
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		// The first `goroutines` hits are rate limited (immediate retry), the rest succeed.
		if n <= goroutines {
			w.Header().Set("Retry-After", "0")
			writeJSON(w, http.StatusTooManyRequests, `{"message":"rate limited"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"ok":true}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	// Use a real shared limiter to exercise concurrent pacing under -race.
	c.limiter = rate.NewLimiter(rate.Limit(500), DefaultRateBurst)

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, resp, err := c.doRequest(context.Background(), "GET", fmt.Sprintf("/test/%d", i), nil)
			if err != nil {
				errCh <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("goroutine %d got status %d", i, resp.StatusCode)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent request failed: %v", err)
	}
}
