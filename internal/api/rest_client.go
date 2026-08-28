package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultPageSize is the default number of items per page for API requests
	DefaultPageSize = 100
	// DefaultHTTPTimeout is the default timeout for HTTP requests
	DefaultHTTPTimeout = 120 * time.Second

	// DefaultRateLimit is the steady-state request rate (requests/second) shared
	// across all scanner workers. Tune here to respect GitLab's rate limits.
	DefaultRateLimit = 10
	// DefaultRateBurst is the shared token-bucket burst size (matches the worker count).
	DefaultRateBurst = 5
	// DefaultMaxRetries bounds the number of retries for transient failures.
	DefaultMaxRetries = 5
	// DefaultBaseBackoff is the base delay used for exponential backoff.
	DefaultBaseBackoff = 500 * time.Millisecond
	// DefaultMaxBackoff caps any single retry/pacing sleep so a bogus server hint
	// (or clock skew) cannot stall a worker indefinitely.
	DefaultMaxBackoff = 30 * time.Second
)

// GitLabClient interface defines the contract for GitLab API interactions
type GitLabClient interface {
	ListProjects(ctx context.Context, options *ListProjectsOptions) ([]*Project, error)
	GetProject(ctx context.Context, projectID interface{}) (*Project, error)
	GetProjectStatistics(ctx context.Context, projectID interface{}) (*ProjectStatistics, error)
	GetGroupByPath(ctx context.Context, groupPath string) (*Group, error)
}

// ListProjectsOptions contains options for listing projects
type ListProjectsOptions struct {
	GroupID           *int
	Membership        *bool
	Owned             *bool
	Starred           *bool
	Archived          *bool
	Visibility        *string
	OrderBy           *string
	Sort              *string
	Search            *string
	Statistics        *bool
	WithIssues        *bool
	WithMergeRequests *bool
	Page              int
	PerPage           int
}

// RestClient implements GitLabClient using direct REST API calls
type RestClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	// limiter is shared by all workers so they collectively respect the rate limit.
	limiter *rate.Limiter

	// Retry/backoff configuration. Defaulted in NewRestClient and overridable in tests
	// so the suite can use tiny, deterministic durations.
	maxRetries  int
	baseBackoff time.Duration
	maxBackoff  time.Duration
	randFloat   func() float64
}

// NewRestClient creates a new REST API based GitLab client
func NewRestClient(baseURL, token string) (*RestClient, error) {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	return &RestClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
		limiter:     rate.NewLimiter(rate.Limit(DefaultRateLimit), DefaultRateBurst),
		maxRetries:  DefaultMaxRetries,
		baseBackoff: DefaultBaseBackoff,
		maxBackoff:  DefaultMaxBackoff,
		randFloat:   rand.Float64,
	}, nil
}

// encodeProjectID URL-encodes a project ID if it's a string (project path), otherwise converts to string
func (c *RestClient) encodeProjectID(projectID interface{}) string {
	if str, ok := projectID.(string); ok {
		return url.PathEscape(str)
	}
	return fmt.Sprintf("%v", projectID)
}

// doRequest performs an authenticated HTTP request with rate limiting, retries,
// and backoff. It retries on 429 and transient 5xx responses (and transport
// errors), honoring Retry-After / RateLimit-Reset hints, and aborts promptly
// when the context is cancelled.
func (c *RestClient) doRequest(ctx context.Context, method, path string, params url.Values) ([]byte, *http.Response, error) {
	// Build full URL
	apiURL := fmt.Sprintf("%s/api/v4%s", c.baseURL, path)
	if len(params) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Respect the shared rate limit; Wait is context-aware and goroutine-safe.
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, nil, err
		}

		req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("PRIVATE-TOKEN", c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Do not retry if the context was cancelled or timed out.
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt == c.maxRetries {
				break
			}
			if sleepErr := c.sleepWithContext(ctx, c.backoffDuration(attempt)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		// Retryable HTTP status: drain and close this body (so the connection can be
		// reused), record the error, then wait and retry.
		if isRetryableStatus(resp.StatusCode) {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			if attempt == c.maxRetries {
				return body, resp, lastErr
			}
			if sleepErr := c.sleepWithContext(ctx, c.waitForRetry(resp, attempt)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}

		// Non-retryable response: read the body and return (success or hard error).
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, resp, fmt.Errorf("failed to read response body: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return body, resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return body, resp, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("request failed after %d attempts", c.maxRetries+1)
	}
	return nil, nil, lastErr
}

// isRetryableStatus reports whether an HTTP status warrants a retry. 501 and other
// 4xx (besides 429) are treated as permanent.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// sleepWithContext sleeps for d unless the context is cancelled first, in which
// case it returns the context error promptly.
func (c *RestClient) sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// waitForRetry determines how long to wait before the next attempt, honoring
// Retry-After first, then RateLimit-Reset, then computed exponential backoff.
func (c *RestClient) waitForRetry(resp *http.Response, attempt int) time.Duration {
	if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
		return c.clampDuration(d)
	}
	// RateLimit-Reset (epoch seconds) also covers the RateLimit-Remaining: 0 case,
	// where the reset time tells us when the budget refills.
	if d, ok := untilRateLimitReset(resp.Header.Get("RateLimit-Reset")); ok {
		return c.clampDuration(d)
	}
	return c.backoffDuration(attempt)
}

// backoffDuration computes an exponential backoff with full jitter, capped at maxBackoff.
func (c *RestClient) backoffDuration(attempt int) time.Duration {
	backoff := float64(c.baseBackoff) * math.Pow(2, float64(attempt))
	if backoff > float64(c.maxBackoff) {
		backoff = float64(c.maxBackoff)
	}
	jittered := backoff * c.randFloat() // full jitter in [0, backoff)
	return c.clampDuration(time.Duration(jittered))
}

// clampDuration floors a duration at 0 and caps it at maxBackoff.
func (c *RestClient) clampDuration(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > c.maxBackoff {
		return c.maxBackoff
	}
	return d
}

// parseRetryAfter parses a Retry-After header value, supporting both the
// delta-seconds form and the HTTP-date form. Past dates / negative deltas
// collapse to 0.
func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// untilRateLimitReset interprets a RateLimit-Reset header (epoch seconds) as a
// wait duration from now. Past resets collapse to 0.
func untilRateLimitReset(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	epoch, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	d := time.Until(time.Unix(epoch, 0))
	if d < 0 {
		d = 0
	}
	return d, true
}

// ListProjects implements the GET /projects endpoint or GET /groups/:id/projects for group filtering
func (c *RestClient) ListProjects(ctx context.Context, options *ListProjectsOptions) ([]*Project, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(options.Page))
	params.Set("per_page", strconv.Itoa(options.PerPage))

	// Key parameters for getting ALL visible projects
	if options.Statistics != nil && *options.Statistics {
		params.Set("statistics", "true")
	}

	// IMPORTANT: The 'archived' parameter in GitLab API is a FILTER, not an inclusion flag
	// - archived=true means "only return archived projects"
	// - archived=false means "only return non-archived projects"
	// - If not set, returns both archived and non-archived projects
	// So we should NOT set this parameter if we want all projects
	// Only set it if the caller explicitly wants to filter by archived status
	if options.Archived != nil {
		// Only set the parameter if explicitly filtering for archived projects only
		params.Set("archived", strconv.FormatBool(*options.Archived))
	}

	// CRITICAL: Use different endpoint when filtering by group ID
	var endpoint string
	if options.GroupID != nil {
		// Use the Groups API endpoint to list projects in a specific group
		endpoint = fmt.Sprintf("/groups/%d/projects", *options.GroupID)
		// IMPORTANT: Include projects from subgroups as well
		params.Set("include_subgroups", "true")
	} else {
		// Use the general projects endpoint for all visible projects
		endpoint = "/projects"
	}

	body, _, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Parse response
	var rawProjects []map[string]interface{}
	if err := json.Unmarshal(body, &rawProjects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our Project type
	projects := make([]*Project, 0, len(rawProjects))
	for _, raw := range rawProjects {
		project := convertRawProject(raw)
		projects = append(projects, project)
	}

	return projects, nil
}

// convertRawProject converts a raw JSON map to our Project type
func convertRawProject(raw map[string]interface{}) *Project {
	project := &Project{}

	// Basic fields
	if id, ok := raw["id"].(float64); ok {
		project.ID = int(id)
	}
	if name, ok := raw["name"].(string); ok {
		project.Name = name
	}
	if path, ok := raw["path"].(string); ok {
		project.Path = path
	}
	if pathWithNamespace, ok := raw["path_with_namespace"].(string); ok {
		project.PathWithNamespace = pathWithNamespace
	}
	if description, ok := raw["description"].(string); ok {
		project.Description = description
	}
	if defaultBranch, ok := raw["default_branch"].(string); ok {
		project.DefaultBranch = defaultBranch
	}
	if httpURL, ok := raw["http_url_to_repo"].(string); ok {
		project.HTTPURLToRepo = httpURL
	}
	if sshURL, ok := raw["ssh_url_to_repo"].(string); ok {
		project.SSHURLToRepo = sshURL
	}
	if webURL, ok := raw["web_url"].(string); ok {
		project.WebURL = webURL
	}
	if visibility, ok := raw["visibility"].(string); ok {
		project.Visibility = visibility
	}
	if archived, ok := raw["archived"].(bool); ok {
		project.Archived = archived
	}
	if emptyRepo, ok := raw["empty_repo"].(bool); ok {
		project.EmptyRepo = emptyRepo
	}

	// Feature flags
	if issuesEnabled, ok := raw["issues_enabled"].(bool); ok {
		project.IssuesEnabled = issuesEnabled
	}
	if mergeRequestsEnabled, ok := raw["merge_requests_enabled"].(bool); ok {
		project.MergeRequestsEnabled = mergeRequestsEnabled
	}

	// Wiki detection: We'll initially set based on wiki_enabled, but will adjust later based on wiki_size
	if wikiEnabled, ok := raw["wiki_enabled"].(bool); ok {
		project.WikiEnabled = wikiEnabled
	}

	// Note: star_count, forks_count, open_issues_count not in our Project struct

	// Dates
	if createdAt, ok := raw["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			project.CreatedAt = &t
		}
	}
	if lastActivityAt, ok := raw["last_activity_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastActivityAt); err == nil {
			project.LastActivityAt = &t
		}
	}

	// Fork detection
	if forkedFromProject, ok := raw["forked_from_project"]; ok && forkedFromProject != nil {
		project.ForkedFromProject = true
	}

	// Statistics (if included)
	if stats, ok := raw["statistics"].(map[string]interface{}); ok {
		project.Statistics = &ProjectStatistics{}
		if commitCount, ok := stats["commit_count"].(float64); ok {
			project.Statistics.CommitCount = int(commitCount)
		}
		if storageSize, ok := stats["storage_size"].(float64); ok {
			project.Statistics.StorageSize = int64(storageSize)
		}
		if repositorySize, ok := stats["repository_size"].(float64); ok {
			project.Statistics.RepositorySize = int64(repositorySize)
		}
		if wikiSize, ok := stats["wiki_size"].(float64); ok {
			project.Statistics.WikiSize = int64(wikiSize)
			// Note: wiki_size might be 0 even if wiki pages exist (GitLab doesn't always
			// update this immediately). We'll keep wiki_enabled as the authoritative source
			// unless we explicitly see evidence that wiki is disabled
		}
		if lfsObjectsSize, ok := stats["lfs_objects_size"].(float64); ok {
			project.Statistics.LFSObjectsSize = int64(lfsObjectsSize)
		}
		if jobArtifactsSize, ok := stats["job_artifacts_size"].(float64); ok {
			project.Statistics.JobArtifactsSize = int64(jobArtifactsSize)
		}
	} else {
		// Initialize empty statistics if not present
		project.Statistics = &ProjectStatistics{}
	}

	// Extract issue and MR counts from top-level project fields
	// These are NOT in the statistics object but are available at the project level
	if openIssuesCount, ok := raw["open_issues_count"].(float64); ok {
		project.Statistics.IssueCount = int(openIssuesCount)
	}

	// GitLab doesn't provide merge_request_count at the project level directly
	// We need to make a separate API call to get the accurate count
	// For now, we'll leave it as 0 and handle it separately if needed

	return project
}

// GetProject implements the GET /projects/:id endpoint with statistics
func (c *RestClient) GetProject(ctx context.Context, projectID interface{}) (*Project, error) {
	params := url.Values{}
	params.Set("statistics", "true")

	encodedProjectID := c.encodeProjectID(projectID)
	path := fmt.Sprintf("/projects/%s", encodedProjectID)
	body, _, err := c.doRequest(ctx, "GET", path, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse project response: %w", err)
	}

	return convertRawProject(raw), nil
}

func (c *RestClient) GetProjectStatistics(ctx context.Context, projectID interface{}) (*ProjectStatistics, error) {
	// In GitLab API, statistics are included when you get a project with statistics=true
	// So we'll fetch the project and return its statistics
	project, err := c.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if project.Statistics == nil {
		// Return empty statistics if not available
		project.Statistics = &ProjectStatistics{}
	}

	// Get additional statistics that aren't included in the basic project response
	// These require separate API calls

	// Get merge request count
	mrCount, err := c.getMergeRequestCount(ctx, projectID)
	if err == nil {
		project.Statistics.MergeRequestCount = mrCount
	} else {
		log.Printf("Warning: Failed to get MR count for project %v: %v", projectID, err)
	}

	// Get branch count
	branchCount, err := c.getBranchCount(ctx, projectID)
	if err == nil {
		project.Statistics.BranchCount = branchCount
	} else {
		log.Printf("Warning: Failed to get branch count for project %v: %v", projectID, err)
	}

	// Get tag count
	tagCount, err := c.getTagCount(ctx, projectID)
	if err == nil {
		project.Statistics.TagCount = tagCount
	} else {
		log.Printf("Warning: Failed to get tag count for project %v: %v", projectID, err)
	}

	// Get member count
	memberCount, err := c.getMemberCount(ctx, projectID)
	if err == nil {
		project.Statistics.MemberCount = memberCount
	} else {
		log.Printf("Warning: Failed to get member count for project %v: %v", projectID, err)
	}

	// Get milestone count
	milestoneCount, err := c.getMilestoneCount(ctx, projectID)
	if err == nil {
		project.Statistics.MilestoneCount = milestoneCount
	} else {
		log.Printf("Warning: Failed to get milestone count for project %v: %v", projectID, err)
	}

	// Get release count
	releaseCount, err := c.getReleaseCount(ctx, projectID)
	if err == nil {
		project.Statistics.ReleaseCount = releaseCount
	} else {
		log.Printf("Warning: Failed to get release count for project %v: %v", projectID, err)
	}

	// Get real protected-branch count from the /protected_branches endpoint
	protectedBranchCount, err := c.getProtectedBranchCount(ctx, projectID)
	if err == nil {
		project.Statistics.ProtectedBranchCount = protectedBranchCount
	} else {
		log.Printf("Warning: Failed to get protected branch count for project %v: %v", projectID, err)
	}

	// Get issue count across all states, overriding the open-only value that
	// convertRawProject derived from open_issues_count.
	issueCount, err := c.getIssueCount(ctx, projectID)
	if err == nil {
		project.Statistics.IssueCount = issueCount
	} else {
		log.Printf("Warning: Failed to get issue count for project %v: %v; falling back to open-only issue count", projectID, err)
	}

	// Check if wiki actually has pages (only if wiki is enabled in settings)
	if project.WikiEnabled {
		project.Statistics.HasWikiPages = c.hasWikiPages(ctx, projectID)
	} else {
		project.Statistics.HasWikiPages = false
	}

	// Get comment counts and review counts (these are more expensive operations)
	// Get merge request review count
	mrReviewCount, err := c.getMergeRequestReviewCount(ctx, projectID)
	if err == nil {
		project.Statistics.MergeRequestReviewCount = mrReviewCount
	} else {
		log.Printf("Warning: Failed to get MR review count for project %v: %v", projectID, err)
	}

	// Get merge request comment count
	mrCommentCount, err := c.getMergeRequestCommentCount(ctx, projectID)
	if err == nil {
		project.Statistics.MergeRequestCommentCount = mrCommentCount
	} else {
		log.Printf("Warning: Failed to get MR comment count for project %v: %v", projectID, err)
	}

	// Get issue comment count
	issueCommentCount, err := c.getIssueCommentCount(ctx, projectID)
	if err == nil {
		project.Statistics.IssueCommentCount = issueCommentCount
	} else {
		log.Printf("Warning: Failed to get issue comment count for project %v: %v", projectID, err)
	}

	return project.Statistics, nil
}

// getCountFromHeader makes a minimal API request and reads the count from the
// X-Total header. It returns a tri-state:
//   - hadHeader=true, err=nil: X-Total was present and numeric -> count is authoritative.
//   - hadHeader=false, err=nil: X-Total was absent or non-numeric -> caller should
//     fall back to counting via full pagination. (GitLab omits X-Total on result
//     sets larger than 10,000 rows, so this is NOT an error and NOT a real 0.)
//   - err!=nil: a transport/HTTP error occurred after retries.
func (c *RestClient) getCountFromHeader(ctx context.Context, endpoint string, extraParams url.Values) (count int, hadHeader bool, err error) {
	params := url.Values{}
	params.Set("per_page", "1")
	params.Set("page", "1")
	for key, values := range extraParams {
		for _, value := range values {
			params.Add(key, value)
		}
	}

	_, resp, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return 0, false, err
	}

	totalHeader := strings.TrimSpace(resp.Header.Get("X-Total"))
	if totalHeader == "" {
		return 0, false, nil
	}
	total, convErr := strconv.Atoi(totalHeader)
	if convErr != nil {
		// Non-numeric header: treat as absent and fall back to pagination.
		return 0, false, nil
	}
	return total, true, nil
}

// countByPagination counts items in a collection by paging through it with
// per_page=100 until a short or empty page is returned. It is intentionally
// unbounded, since the X-Total header is omitted exactly on the largest result
// sets; the shared rate limiter and context keep it safe.
func (c *RestClient) countByPagination(ctx context.Context, endpoint string, extraParams url.Values) (int, error) {
	total := 0
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("per_page", strconv.Itoa(DefaultPageSize))
		params.Set("page", strconv.Itoa(page))
		for key, values := range extraParams {
			for _, value := range values {
				params.Add(key, value)
			}
		}

		body, _, err := c.doRequest(ctx, "GET", endpoint, params)
		if err != nil {
			return 0, err
		}

		var items []json.RawMessage
		if err := json.Unmarshal(body, &items); err != nil {
			return 0, fmt.Errorf("failed to parse pagination response: %w", err)
		}

		total += len(items)
		if len(items) < DefaultPageSize {
			break
		}
	}
	return total, nil
}

// countCollection returns the size of a collection, preferring the X-Total header
// and falling back to full pagination when the header is absent.
func (c *RestClient) countCollection(ctx context.Context, endpoint string, extraParams url.Values) (int, error) {
	count, hadHeader, err := c.getCountFromHeader(ctx, endpoint, extraParams)
	if err != nil {
		return 0, err
	}
	if hadHeader {
		return count, nil
	}
	return c.countByPagination(ctx, endpoint, extraParams)
}

// getMergeRequestCount gets the total count of merge requests for a project
func (c *RestClient) getMergeRequestCount(ctx context.Context, projectID interface{}) (int, error) {
	params := url.Values{}
	params.Set("scope", "all")

	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/merge_requests", encodedProjectID)
	return c.countCollection(ctx, endpoint, params)
}

// getBranchCount gets the total count of branches for a project
func (c *RestClient) getBranchCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/repository/branches", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// getTagCount gets the total count of tags for a project
func (c *RestClient) getTagCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/repository/tags", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// getMemberCount gets the total count of members for a project
func (c *RestClient) getMemberCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/members/all", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// getMilestoneCount gets the total count of milestones for a project
func (c *RestClient) getMilestoneCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/milestones", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// getReleaseCount gets the total count of releases for a project
func (c *RestClient) getReleaseCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/releases", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// getProtectedBranchCount gets the real count of protected branches for a project.
// The protected-branches list is small, so we count via full pagination directly
// instead of countCollection to skip the wasted per_page=1 X-Total probe request.
func (c *RestClient) getProtectedBranchCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/protected_branches", encodedProjectID)
	return c.countByPagination(ctx, endpoint, nil)
}

// getIssueCount gets the total count of issues (all states) for a project.
// GitLab's issues list returns all states by default, so we pass no state filter.
func (c *RestClient) getIssueCount(ctx context.Context, projectID interface{}) (int, error) {
	encodedProjectID := c.encodeProjectID(projectID)
	endpoint := fmt.Sprintf("/projects/%s/issues", encodedProjectID)
	return c.countCollection(ctx, endpoint, nil)
}

// hasWikiPages checks if a project actually has wiki pages
func (c *RestClient) hasWikiPages(ctx context.Context, projectID interface{}) bool {
	params := url.Values{}
	params.Set("per_page", "1")
	params.Set("page", "1")

	encodedProjectID := c.encodeProjectID(projectID)
	path := fmt.Sprintf("/projects/%s/wikis", encodedProjectID)
	body, _, err := c.doRequest(ctx, "GET", path, params)
	if err != nil {
		// If we get an error, assume no wiki (could be disabled or no access)
		return false
	}

	// Parse the response to see if there are any wiki pages
	var wikis []map[string]interface{}
	if err := json.Unmarshal(body, &wikis); err != nil {
		return false
	}

	return len(wikis) > 0
}

// getMergeRequestReviewCount gets the total number of approver entries across all
// merge requests in a project. It paginates through every MR (scope=all) and sums
// len(approved_by) for each — i.e., an approval-entry total, not a count of MRs that
// have at least one approval. Upvotes/reactions are intentionally excluded.
func (c *RestClient) getMergeRequestReviewCount(ctx context.Context, projectID interface{}) (int, error) {
	totalReviews := 0
	mrParams := url.Values{}
	mrParams.Set("scope", "all")
	mrParams.Set("per_page", strconv.Itoa(DefaultPageSize))

	encodedProjectID := c.encodeProjectID(projectID)
	mrPath := fmt.Sprintf("/projects/%s/merge_requests", encodedProjectID)
	for page := 1; ; page++ {
		mrParams.Set("page", strconv.Itoa(page))
		mrBody, resp, err := c.doRequest(ctx, "GET", mrPath, mrParams)
		if err != nil {
			log.Printf("Warning: mr_review_count for project %v is PARTIAL: failed on page %d: %v", projectID, page, err)
			return totalReviews, nil
		}

		var pageMRs []map[string]interface{}
		if err := json.Unmarshal(mrBody, &pageMRs); err != nil {
			log.Printf("Warning: mr_review_count for project %v is PARTIAL: failed to parse page %d: %v", projectID, page, err)
			return totalReviews, nil
		}

		for _, mr := range pageMRs {
			// Only count actual approvals from approved_by, not upvotes
			// Upvotes are just "thumbs up" reactions, not actual code reviews
			if approvers, ok := mr["approved_by"].([]interface{}); ok {
				totalReviews += len(approvers)
			}
		}

		if isLastPage(resp, len(pageMRs)) {
			break
		}
	}

	return totalReviews, nil
}

// isLastPage reports whether pagination should stop after the current page.
// It follows GitLab's documented signal — an empty/absent X-Next-Page header means
// there is no next page — and additionally stops on a short page (fewer than
// DefaultPageSize items), which is always the last page. Either signal terminates.
func isLastPage(resp *http.Response, pageItems int) bool {
	if resp != nil {
		if next := strings.TrimSpace(resp.Header.Get("X-Next-Page")); next == "" {
			return true
		}
	}
	return pageItems < DefaultPageSize
}

// getMergeRequestCommentCount gets the total count of comments on merge requests
func (c *RestClient) getMergeRequestCommentCount(ctx context.Context, projectID interface{}) (int, error) {
	// In GitLab, MR comments are called "notes" and include both regular comments and code review comments.
	// We sum the user_notes_count field across all merge requests (scope=all), paginating to exhaustion.
	encodedProjectID := c.encodeProjectID(projectID)

	totalNotes := 0
	mrParams := url.Values{}
	mrParams.Set("scope", "all")
	mrParams.Set("per_page", strconv.Itoa(DefaultPageSize))
	mrPath := fmt.Sprintf("/projects/%s/merge_requests", encodedProjectID)
	for page := 1; ; page++ {
		mrParams.Set("page", strconv.Itoa(page))
		mrBody, resp, err := c.doRequest(ctx, "GET", mrPath, mrParams)
		if err != nil {
			log.Printf("Warning: mr_comment_count for project %v is PARTIAL: failed on page %d: %v", projectID, page, err)
			return totalNotes, nil
		}

		var pageMRs []map[string]interface{}
		if err := json.Unmarshal(mrBody, &pageMRs); err != nil {
			log.Printf("Warning: mr_comment_count for project %v is PARTIAL: failed to parse page %d: %v", projectID, page, err)
			return totalNotes, nil
		}

		for _, mr := range pageMRs {
			if userNotesCount, ok := mr["user_notes_count"].(float64); ok {
				totalNotes += int(userNotesCount)
			}
		}

		if isLastPage(resp, len(pageMRs)) {
			break
		}
	}

	return totalNotes, nil
}

// getIssueCommentCount gets the total count of comments on issues
func (c *RestClient) getIssueCommentCount(ctx context.Context, projectID interface{}) (int, error) {
	// Similar to MR comments, we fetch issues (scope=all) and sum their user_notes_count,
	// paginating to exhaustion.
	totalNotes := 0
	issueParams := url.Values{}
	issueParams.Set("scope", "all")
	issueParams.Set("per_page", strconv.Itoa(DefaultPageSize))

	encodedProjectID := c.encodeProjectID(projectID)
	issuePath := fmt.Sprintf("/projects/%s/issues", encodedProjectID)
	for page := 1; ; page++ {
		issueParams.Set("page", strconv.Itoa(page))
		issueBody, resp, err := c.doRequest(ctx, "GET", issuePath, issueParams)
		if err != nil {
			log.Printf("Warning: issue_comment_count for project %v is PARTIAL: failed on page %d: %v", projectID, page, err)
			return totalNotes, nil
		}

		var pageIssues []map[string]interface{}
		if err := json.Unmarshal(issueBody, &pageIssues); err != nil {
			log.Printf("Warning: issue_comment_count for project %v is PARTIAL: failed to parse page %d: %v", projectID, page, err)
			return totalNotes, nil
		}

		for _, issue := range pageIssues {
			if userNotesCount, ok := issue["user_notes_count"].(float64); ok {
				totalNotes += int(userNotesCount)
			}
		}

		if isLastPage(resp, len(pageIssues)) {
			break
		}
	}

	return totalNotes, nil
}

// GetGroupByPath retrieves a group by its full path (e.g., "mygroup" or "mygroup/subgroup")
// This is used to resolve namespace names to group IDs for efficient filtering
func (c *RestClient) GetGroupByPath(ctx context.Context, groupPath string) (*Group, error) {
	// URL-encode the group path (GitLab API requires path to be URL-encoded)
	encodedPath := url.PathEscape(groupPath)
	path := fmt.Sprintf("/groups/%s", encodedPath)

	body, _, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get group %s: %w", groupPath, err)
	}

	var group Group
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("failed to parse group response: %w", err)
	}

	return &group, nil
}
