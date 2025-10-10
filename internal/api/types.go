package api

import "time"

// Project represents a GitLab project
type Project struct {
	ID                   int                `json:"id"`
	Name                 string             `json:"name"`
	Path                 string             `json:"path"`
	PathWithNamespace    string             `json:"path_with_namespace"`
	Description          string             `json:"description"`
	DefaultBranch        string             `json:"default_branch"`
	WebURL               string             `json:"web_url"`
	SSHURLToRepo         string             `json:"ssh_url_to_repo"`
	HTTPURLToRepo        string             `json:"http_url_to_repo"`
	Visibility           string             `json:"visibility"`
	Archived             bool               `json:"archived"`
	IssuesEnabled        bool               `json:"issues_enabled"`
	MergeRequestsEnabled bool               `json:"merge_requests_enabled"`
	WikiEnabled          bool               `json:"wiki_enabled"`
	ForkedFromProject    bool               `json:"forked_from_project"`
	CreatedAt            *time.Time         `json:"created_at"`
	LastActivityAt       *time.Time         `json:"last_activity_at"`
	EmptyRepo            bool               `json:"empty_repo"`
	Statistics           *ProjectStatistics `json:"statistics,omitempty"`
}

// ProjectStatistics represents project statistics from GitLab
type ProjectStatistics struct {
	ProjectID                int    `json:"project_id,omitempty"`
	ProjectName              string `json:"project_name,omitempty"`
	CommitCount              int    `json:"commit_count"`
	StorageSize              int64  `json:"storage_size"`
	RepositorySize           int64  `json:"repository_size"`
	WikiSize                 int64  `json:"wiki_size"`
	LFSObjectsSize           int64  `json:"lfs_objects_size"`
	JobArtifactsSize         int64  `json:"job_artifacts_size"`
	BranchCount              int    `json:"branch_count,omitempty"`
	TagCount                 int    `json:"tag_count,omitempty"`
	MemberCount              int    `json:"member_count,omitempty"`
	IssueCount               int    `json:"issue_count,omitempty"`
	MergeRequestCount        int    `json:"merge_request_count,omitempty"`
	MilestoneCount           int    `json:"milestone_count,omitempty"`
	ReleaseCount             int    `json:"release_count,omitempty"`
	HasWikiPages             bool   `json:"-"` // Whether wiki actually has pages (not from API, computed)
	MergeRequestReviewCount  int    `json:"-"` // Number of MR reviews/approvals (computed)
	MergeRequestCommentCount int    `json:"-"` // Total comments on merge requests (computed)
	IssueCommentCount        int    `json:"-"` // Total comments on issues (computed)
	CommitCommentCount       int    `json:"-"` // Total comments on commits (computed)
}

// Branch represents a GitLab branch
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

// Tag represents a GitLab tag
type Tag struct {
	Name               string `json:"name"`
	Message            string `json:"message"`
	ReleaseDescription string `json:"release_description"`
}

// Member represents a GitLab project member
type Member struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	AccessLevel int    `json:"access_level"`
}

// Issue represents a GitLab issue
type Issue struct {
	ID    int    `json:"id"`
	IID   int    `json:"iid"`
	Title string `json:"title"`
	State string `json:"state"`
}

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	ID    int    `json:"id"`
	IID   int    `json:"iid"`
	Title string `json:"title"`
	State string `json:"state"`
}

// Milestone represents a GitLab milestone
type Milestone struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
}

// Release represents a GitLab release
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
