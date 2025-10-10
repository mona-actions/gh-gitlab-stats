package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
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
			Timeout: 120 * time.Second, // Increased timeout for large responses
		},
	}, nil
}

// doRequest performs an HTTP request with authentication
func (c *RestClient) doRequest(ctx context.Context, method, path string, params url.Values) ([]byte, *http.Response, error) {
	// Build full URL
	apiURL := fmt.Sprintf("%s/api/v4%s", c.baseURL, path)
	if len(params) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, resp, nil
}

// ListProjects implements the GET /projects endpoint
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

	// IMPORTANT: Don't set membership parameter to get all visible projects
	// The GitLab API defaults to returning all visible projects when membership is not specified

	body, _, err := c.doRequest(ctx, "GET", "/projects", params)
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

	path := fmt.Sprintf("/projects/%v", projectID)
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
	}

	// Get branch count
	branchCount, err := c.getBranchCount(ctx, projectID)
	if err == nil {
		project.Statistics.BranchCount = branchCount
	}

	// Get tag count
	tagCount, err := c.getTagCount(ctx, projectID)
	if err == nil {
		project.Statistics.TagCount = tagCount
	}

	// Get member count
	memberCount, err := c.getMemberCount(ctx, projectID)
	if err == nil {
		project.Statistics.MemberCount = memberCount
	}

	// Get milestone count
	milestoneCount, err := c.getMilestoneCount(ctx, projectID)
	if err == nil {
		project.Statistics.MilestoneCount = milestoneCount
	}

	// Get release count
	releaseCount, err := c.getReleaseCount(ctx, projectID)
	if err == nil {
		project.Statistics.ReleaseCount = releaseCount
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
	}

	// Get merge request comment count
	mrCommentCount, err := c.getMergeRequestCommentCount(ctx, projectID)
	if err == nil {
		project.Statistics.MergeRequestCommentCount = mrCommentCount
	}

	// Get issue comment count
	issueCommentCount, err := c.getIssueCommentCount(ctx, projectID)
	if err == nil {
		project.Statistics.IssueCommentCount = issueCommentCount
	}

	return project.Statistics, nil
}

// getCountFromHeader makes a minimal API request and returns the count from X-Total header
func (c *RestClient) getCountFromHeader(ctx context.Context, endpoint string, extraParams url.Values) int {
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
		return 0
	}

	if totalHeader := resp.Header.Get("X-Total"); totalHeader != "" {
		if total, err := strconv.Atoi(totalHeader); err == nil {
			return total
		}
	}
	return 0
}

// getMergeRequestCount gets the total count of merge requests for a project
func (c *RestClient) getMergeRequestCount(ctx context.Context, projectID interface{}) (int, error) {
	params := url.Values{}
	params.Set("scope", "all")
	endpoint := fmt.Sprintf("/projects/%v/merge_requests", projectID)
	return c.getCountFromHeader(ctx, endpoint, params), nil
}

// getBranchCount gets the total count of branches for a project
func (c *RestClient) getBranchCount(ctx context.Context, projectID interface{}) (int, error) {
	endpoint := fmt.Sprintf("/projects/%v/repository/branches", projectID)
	return c.getCountFromHeader(ctx, endpoint, nil), nil
}

// getTagCount gets the total count of tags for a project
func (c *RestClient) getTagCount(ctx context.Context, projectID interface{}) (int, error) {
	endpoint := fmt.Sprintf("/projects/%v/repository/tags", projectID)
	return c.getCountFromHeader(ctx, endpoint, nil), nil
}

// getMemberCount gets the total count of members for a project
func (c *RestClient) getMemberCount(ctx context.Context, projectID interface{}) (int, error) {
	endpoint := fmt.Sprintf("/projects/%v/members/all", projectID)
	return c.getCountFromHeader(ctx, endpoint, nil), nil
}

// getMilestoneCount gets the total count of milestones for a project
func (c *RestClient) getMilestoneCount(ctx context.Context, projectID interface{}) (int, error) {
	endpoint := fmt.Sprintf("/projects/%v/milestones", projectID)
	return c.getCountFromHeader(ctx, endpoint, nil), nil
}

// getReleaseCount gets the total count of releases for a project
func (c *RestClient) getReleaseCount(ctx context.Context, projectID interface{}) (int, error) {
	endpoint := fmt.Sprintf("/projects/%v/releases", projectID)
	return c.getCountFromHeader(ctx, endpoint, nil), nil
}

// hasWikiPages checks if a project actually has wiki pages
func (c *RestClient) hasWikiPages(ctx context.Context, projectID interface{}) bool {
	params := url.Values{}
	params.Set("per_page", "1")
	params.Set("page", "1")

	path := fmt.Sprintf("/projects/%v/wikis", projectID)
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

// getMergeRequestReviewCount gets the total count of MR approvals/reviews
func (c *RestClient) getMergeRequestReviewCount(ctx context.Context, projectID interface{}) (int, error) {
	// In GitLab, reviews are tracked as "approvals" on merge requests
	// We'll count MRs that have at least one approval
	totalReviews := 0
	mrParams := url.Values{}
	mrParams.Set("scope", "all")
	mrParams.Set("per_page", "100")

	for page := 1; page <= 10; page++ { // Limit to first 1000 MRs
		mrParams.Set("page", strconv.Itoa(page))
		mrPath := fmt.Sprintf("/projects/%v/merge_requests", projectID)
		mrBody, _, err := c.doRequest(ctx, "GET", mrPath, mrParams)
		if err != nil {
			break
		}

		var pageMRs []map[string]interface{}
		if err := json.Unmarshal(mrBody, &pageMRs); err != nil {
			break
		}

		if len(pageMRs) == 0 {
			break
		}

		for _, mr := range pageMRs {
			// Only count actual approvals from approved_by, not upvotes
			// Upvotes are just "thumbs up" reactions, not actual code reviews
			if approvers, ok := mr["approved_by"].([]interface{}); ok {
				totalReviews += len(approvers)
			}
		}

		if len(pageMRs) < 100 {
			break
		}
	}

	return totalReviews, nil
}

// getMergeRequestCommentCount gets the total count of comments on merge requests
func (c *RestClient) getMergeRequestCommentCount(ctx context.Context, projectID interface{}) (int, error) {
	// In GitLab, MR comments are called "notes" and include both regular comments and code review comments
	// We need to get notes from the merge_requests endpoint
	params := url.Values{}
	params.Set("scope", "all")
	params.Set("per_page", "1")
	params.Set("page", "1")

	path := fmt.Sprintf("/projects/%v/merge_requests", projectID)
	body, _, err := c.doRequest(ctx, "GET", path, params)
	if err != nil {
		return 0, err
	}

	// Parse merge requests to get their IDs, then count notes
	var mrs []map[string]interface{}
	if err := json.Unmarshal(body, &mrs); err != nil {
		return 0, err
	}

	// For now, we'll use the user_notes_count field from MRs
	// This requires fetching all MRs to sum up the notes
	// To avoid excessive API calls, we'll get the first page and estimate
	totalNotes := 0
	mrParams := url.Values{}
	mrParams.Set("scope", "all")
	mrParams.Set("per_page", "100")

	for page := 1; page <= 10; page++ { // Limit to first 1000 MRs
		mrParams.Set("page", strconv.Itoa(page))
		mrPath := fmt.Sprintf("/projects/%v/merge_requests", projectID)
		mrBody, _, err := c.doRequest(ctx, "GET", mrPath, mrParams)
		if err != nil {
			break
		}

		var pageMRs []map[string]interface{}
		if err := json.Unmarshal(mrBody, &pageMRs); err != nil {
			break
		}

		if len(pageMRs) == 0 {
			break
		}

		for _, mr := range pageMRs {
			if userNotesCount, ok := mr["user_notes_count"].(float64); ok {
				totalNotes += int(userNotesCount)
			}
		}

		if len(pageMRs) < 100 {
			break
		}
	}

	return totalNotes, nil
}

// getIssueCommentCount gets the total count of comments on issues
func (c *RestClient) getIssueCommentCount(ctx context.Context, projectID interface{}) (int, error) {
	// Similar to MR comments, we need to fetch issues and sum up their notes
	totalNotes := 0
	issueParams := url.Values{}
	issueParams.Set("scope", "all")
	issueParams.Set("per_page", "100")

	for page := 1; page <= 10; page++ { // Limit to first 1000 issues
		issueParams.Set("page", strconv.Itoa(page))
		issuePath := fmt.Sprintf("/projects/%v/issues", projectID)
		issueBody, _, err := c.doRequest(ctx, "GET", issuePath, issueParams)
		if err != nil {
			break
		}

		var pageIssues []map[string]interface{}
		if err := json.Unmarshal(issueBody, &pageIssues); err != nil {
			break
		}

		if len(pageIssues) == 0 {
			break
		}

		for _, issue := range pageIssues {
			if userNotesCount, ok := issue["user_notes_count"].(float64); ok {
				totalNotes += int(userNotesCount)
			}
		}

		if len(pageIssues) < 100 {
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
