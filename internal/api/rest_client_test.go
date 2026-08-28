package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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

func TestGetProtectedBranchCount_FullPagination(t *testing.T) {
	// countByPagination is used directly: page 1 full (100), page 2 partial (30) => 130.
	// The per_page=1 probe is never hit, so probeHeader is irrelevant here.
	srv := httptest.NewServer(paginationHandler("", map[string]int{"1": 100, "2": 30}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getProtectedBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 130 {
		t.Fatalf("expected 130 from full pagination, got %d", count)
	}
}

func TestGetProtectedBranchCount_EmptyIsRealZero(t *testing.T) {
	// No protected branches: the first page is empty and short, so the count is a real 0.
	srv := httptest.NewServer(paginationHandler("", map[string]int{}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getProtectedBranchCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 for an empty protected-branches list, got %d", count)
	}
}

func TestGetIssueCount_PresentXTotalAllStates(t *testing.T) {
	// GitLab returns all states by default, so getIssueCount must not send a state filter.
	var sawStateParam bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "" {
			sawStateParam = true
		}
		// Present, numeric X-Total on the probe -> authoritative all-states count.
		paginationHandler("42", map[string]int{})(w, r)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getIssueCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected 42 from X-Total, got %d", count)
	}
	if sawStateParam {
		t.Fatal("getIssueCount must not send a state filter (should count all states)")
	}
}

func TestGetIssueCount_MissingXTotalFallsBackToPagination(t *testing.T) {
	// GitLab omits X-Total on >10k-row sets: fall back to pagination (100 + 50 = 150).
	srv := httptest.NewServer(paginationHandler("", map[string]int{"1": 100, "2": 50}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	count, err := c.getIssueCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 150 {
		t.Fatalf("expected 150 from pagination fallback, got %d", count)
	}
}

// projectStatsHandler routes the many endpoints GetProjectStatistics touches. The
// project GET returns open_issues_count=3 (open-only), while /issues reports 7 all
// states and /protected_branches returns 2 entries, so the test can assert the
// override and the real protected-branch count.
func projectStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(path, "/issues"):
			// getIssueCount probe (per_page=1) reports 7 via X-Total; the
			// per_page=100 issue-comment loop just gets an empty page.
			if q.Get("per_page") == "1" {
				w.Header().Set("X-Total", "7")
			}
			writeJSON(w, http.StatusOK, "[]")
		case strings.HasSuffix(path, "/protected_branches"):
			// Full-pagination count: 2 entries on page 1, then a short page ends it.
			if q.Get("page") == "1" {
				writeJSON(w, http.StatusOK, "[{},{}]")
				return
			}
			writeJSON(w, http.StatusOK, "[]")
		case strings.HasSuffix(path, "/merge_requests"):
			// MR count probe + review/comment loops: no data needed.
			if q.Get("per_page") == "1" {
				w.Header().Set("X-Total", "0")
			}
			writeJSON(w, http.StatusOK, "[]")
		case strings.HasSuffix(path, "/projects/1"):
			// The project GET with statistics=true. Open-only issue count is 3.
			writeJSON(w, http.StatusOK, `{"id":1,"name":"proj","wiki_enabled":false,"open_issues_count":3,"statistics":{"commit_count":50,"repository_size":1024}}`)
		default:
			// Every other counter (branches, tags, members, milestones, releases).
			if q.Get("per_page") == "1" {
				w.Header().Set("X-Total", "0")
			}
			writeJSON(w, http.StatusOK, "[]")
		}
	}
}

func TestGetProjectStatistics_OverridesIssueCountAndRealProtectedBranches(t *testing.T) {
	srv := httptest.NewServer(projectStatsHandler())
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	stats, err := c.GetProjectStatistics(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// open_issues_count was 3; the all-states /issues count (7) must override it.
	if stats.IssueCount != 7 {
		t.Fatalf("expected IssueCount overridden to 7 (all states), got %d", stats.IssueCount)
	}
	// Protected branches come from the real endpoint, not a branch-count heuristic.
	if stats.ProtectedBranchCount != 2 {
		t.Fatalf("expected ProtectedBranchCount 2 from /protected_branches, got %d", stats.ProtectedBranchCount)
	}
}

// aggHandler serves paginated list pages for the aggregation loops
// (getMergeRequestReviewCount / getMergeRequestCommentCount / getIssueCommentCount).
// pageItems maps a 1-based page number to the number of items returned. While the
// current page is below lastPage, an X-Next-Page header is set to drive the loop the
// way GitLab does; on/after lastPage the header is absent. Any page in failPages
// returns HTTP 500 (persistently, to exhaust the retry budget). field selects the
// summed JSON attribute: "approved_by" emits a single-approver array per item,
// anything else emits that numeric field set to 1 per item.
func aggHandler(field string, pageItems map[int]int, lastPage int, failPages map[int]bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if failPages[page] {
			writeJSON(w, http.StatusInternalServerError, `{"error":"boom"}`)
			return
		}
		if page < lastPage {
			w.Header().Set("X-Next-Page", strconv.Itoa(page+1))
		}
		n := pageItems[page]
		items := make([]string, n)
		for i := range items {
			if field == "approved_by" {
				items[i] = `{"approved_by":[{"id":1}]}`
			} else {
				items[i] = `{"` + field + `":1}`
			}
		}
		writeJSON(w, http.StatusOK, "["+join(items, ",")+"]")
	}
}

// TestGetMergeRequestReviewCount_ExhaustsBeyond1000 proves the old 1,000-item cap is
// gone: 12 full pages (100 each) plus a 13th short page must all be summed.
func TestGetMergeRequestReviewCount_ExhaustsBeyond1000(t *testing.T) {
	pages := map[int]int{13: 50}
	for p := 1; p <= 12; p++ {
		pages[p] = 100
	}
	srv := httptest.NewServer(aggHandler("approved_by", pages, 13, nil))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	got, err := c.getMergeRequestReviewCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1250 {
		t.Fatalf("expected 1250 approver entries across 13 pages, got %d", got)
	}
}

// TestGetIssueCommentCount_ExactMultipleBoundary proves no off-by-one / no premature
// stop when the total is an exact multiple of the page size: two full pages, then an
// absent X-Next-Page on the last full page ends pagination.
func TestGetIssueCommentCount_ExactMultipleBoundary(t *testing.T) {
	srv := httptest.NewServer(aggHandler("user_notes_count", map[int]int{1: 100, 2: 100}, 2, nil))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	got, err := c.getIssueCommentCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 200 {
		t.Fatalf("expected 200 notes across 2 full pages, got %d", got)
	}
}

// TestGetMergeRequestCommentCount_PartialOnFailureReturnsNilErr verifies the
// partial-safety contract: a mid-pagination request failure (retries exhausted) stops
// the loop and returns the sum accumulated so far with a nil error.
func TestGetMergeRequestCommentCount_PartialOnFailureReturnsNilErr(t *testing.T) {
	srv := httptest.NewServer(aggHandler("user_notes_count", map[int]int{1: 100, 2: 100}, 5, map[int]bool{3: true}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	got, err := c.getMergeRequestCommentCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error on partial count, got %v", err)
	}
	if got != 200 {
		t.Fatalf("expected partial sum 200 from first two pages, got %d", got)
	}
}

// TestGetIssueCommentCount_ZeroItemsNoWarning confirms an empty first page returns 0
// with a nil error and emits no PARTIAL warning.
func TestGetIssueCommentCount_ZeroItemsNoWarning(t *testing.T) {
	srv := httptest.NewServer(aggHandler("user_notes_count", map[int]int{1: 0}, 1, nil))
	defer srv.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	c := newTestClient(t, srv.URL)
	got, err := c.getIssueCommentCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Fatalf("expected 0 for empty project, got %d", got)
	}
	if strings.Contains(buf.String(), "PARTIAL") {
		t.Fatalf("did not expect a PARTIAL warning, got: %q", buf.String())
	}
}

// TestGetIssueCommentCount_PartialEmitsWarning asserts that a mid-pagination failure
// surfaces the durable PARTIAL warning naming the project and field. This test mutates
// the global logger, so it must not run in parallel.
func TestGetIssueCommentCount_PartialEmitsWarning(t *testing.T) {
	srv := httptest.NewServer(aggHandler("user_notes_count", map[int]int{1: 100}, 5, map[int]bool{2: true}))
	defer srv.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	c := newTestClient(t, srv.URL)
	got, err := c.getIssueCommentCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error on partial count, got %v", err)
	}
	if got != 100 {
		t.Fatalf("expected partial sum 100, got %d", got)
	}
	if !strings.Contains(buf.String(), "issue_comment_count") || !strings.Contains(buf.String(), "PARTIAL") {
		t.Fatalf("expected PARTIAL warning for issue_comment_count, got: %q", buf.String())
	}
}
