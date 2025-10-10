package models

import "time"

// RepositoryStats represents the CSV output structure matching the GitHub version
type RepositoryStats struct {
	Namespace            string     `csv:"Namespace"`
	RepoName             string     `csv:"Repo_Name"`
	IsEmpty              bool       `csv:"Is_Empty"`
	IsFork               bool       `csv:"isFork"`
	IsArchive            bool       `csv:"isArchive"`
	RepoSizeMB           float64    `csv:"Repo_Size(mb)"`
	LFSSizeMB            float64    `csv:"LFS_Size(mb)"`
	RecordCount          int        `csv:"Record_Count"`
	CollaboratorCount    int        `csv:"Collaborator_Count"`
	ProtectedBranchCount int        `csv:"Protected_Branch_Count"`
	PRReviewCount        int        `csv:"PR_Review_Count"`
	MilestoneCount       int        `csv:"Milestone_Count"`
	IssueCount           int        `csv:"Issue_Count"`
	PRCount              int        `csv:"PR_Count"`
	PRReviewCommentCount int        `csv:"PR_Review_Comment_Count"`
	CommitCount          int        `csv:"Commit_Count"`
	IssueCommentCount    int        `csv:"Issue_Comment_Count"`
	ReleaseCount         int        `csv:"Release_Count"`
	BranchCount          int        `csv:"Branch_Count"`
	TagCount             int        `csv:"Tag_Count"`
	HasWiki              bool       `csv:"Has_Wiki"`
	FullURL              string     `csv:"Full_URL"`
	Created              *time.Time `csv:"Created"`
	LastPush             *time.Time `csv:"Last_Push"`
	LastUpdate           *time.Time `csv:"Last_Update"`
}

// ScanOptions represents the options for scanning GitLab
type ScanOptions struct {
	GitLabURL       string
	Token           string
	GroupID         *int
	OutputFormat    string
	OutputFile      string
	Verbose         bool
	IncludeArchived bool
	MaxProjects     int
}

// ScanResult represents the result of a GitLab scan operation
type ScanResult struct {
	TotalProjects     int
	ProcessedProjects int
	RepositoryStats   []*RepositoryStats
	Errors            []error
	Duration          time.Duration
}
