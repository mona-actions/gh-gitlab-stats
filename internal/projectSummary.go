package internal

import (
	"log"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/api/commits"
	"github.com/mona-actions/gh-gitlab-stats/api/issues"
	"github.com/mona-actions/gh-gitlab-stats/api/members"
	"github.com/mona-actions/gh-gitlab-stats/api/mergerequests"
	"github.com/mona-actions/gh-gitlab-stats/api/projects"
	"github.com/pterm/pterm"
	"github.com/xanzy/go-gitlab"
)

type ProjectSummary struct {
	Namespace               string
	ProjectName             string
	IsEmpty                 bool
	Last_Push               *time.Time
	Last_Update             *time.Time
	IsFork                  bool
	RepoSize                int64
	RecordCount             int
	CollaboratorCount       int
	ProtectedBranchCount    int
	MergeRequestReviewCount int
	MilestoneCount          int
	IssueCount              int
	MergeRequestCount       int
	MRReviewCommentCount    int
	CommitCommentCount      int
	IssueCommentCount       int
	ReleaseCount            int
	BranchCount             int
	TagCount                int
	DiscussionCount         int
	HasWiki                 bool
	FullUrl                 string
	MigrationIssue          bool
}

var (
	gitlabProjectsSummary []*ProjectSummary
)

func GetProjectSummary(gitlabProjects []*gitlab.Project, client *gitlab.Client) []*ProjectSummary {

	isMigrationIssue := false
	var issueCommentCount int
	var mergeRequestCommentCount int
	var repoSizeInMB int64

	for _, project := range gitlabProjects {
		var protectedBranchesCount int
		repoWithOwner := project.Namespace.Name + "/" + project.Name
		projectSummarySpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching " + repoWithOwner + " MetaData")

		commits := commits.GetCommitActivity(project, client)

		mergeRequests := mergerequests.GetMergeRequests(project, client)

		wikis := projects.GetProjectWikis(project, client)

		for _, mergeRequest := range mergeRequests {
			mergeRequestComments := mergerequests.GetMergeRequestComments(project, mergeRequest, client)
			mergeRequestCommentCount += len(mergeRequestComments)
		}

		projectMembers := members.GetProjectMembers(project, client)

		projectBranches := projects.GetProjectBranches(project, client)
		for _, branch := range projectBranches {
			if branch.Protected {
				protectedBranchesCount++
			}
		}

		projectMilestones := projects.GetProjectMilestones(project, client)

		projectIssues := issues.GetProjectIssues(project, client)

		for _, issue := range projectIssues {
			issueComments := issues.GetIssueComments(project, issue, client)
			issueCommentCount += len(issueComments)
		}

		projectReleases := projects.GetProjectReleases(project, client)

		recordCount := len(commits) + len(projectIssues) + len(mergeRequests) + len(projectMilestones) + len(projectReleases) + len(projectBranches) + len(project.TagList) + mergeRequestCommentCount + issueCommentCount
		if project != nil && project.Statistics != nil {
			repoSizeInMB = (project.Statistics.RepositorySize / 1000000)
		} else {
			log.Println(project, " and/or it's statistics value was found to be nil, repoSize will report 0")
		}
		if recordCount > 60000 || repoSizeInMB > 1500 {
			isMigrationIssue = true
		}
		if recordCount > 60000 || repoSizeInMB > 1500 {
			isMigrationIssue = true
		}
		row := &ProjectSummary{
			Namespace:            project.Namespace.Name,
			ProjectName:          project.Name,
			IsEmpty:              project.EmptyRepo,
			Last_Update:          project.LastActivityAt,
			IsFork:               project.ForkedFromProject != nil,
			RepoSize:             repoSizeInMB,
			RecordCount:          recordCount,
			CollaboratorCount:    len(projectMembers),
			ProtectedBranchCount: protectedBranchesCount,
			//MergeRequestReviewCount:
			MilestoneCount:       len(projectMilestones),
			IssueCount:           len(projectIssues),
			MergeRequestCount:    len(mergeRequests),
			MRReviewCommentCount: mergeRequestCommentCount,
			CommitCommentCount:   len(commits),
			IssueCommentCount:    issueCommentCount,
			//IssueEventCount:
			ReleaseCount: len(projectReleases),
			BranchCount:  len(projectBranches),
			TagCount:     len(project.TagList),
			//DiscussionCount:
			HasWiki:        len(wikis) > 0,
			FullUrl:        project.WebURL,
			MigrationIssue: isMigrationIssue,
		}
		gitlabProjectsSummary = append(gitlabProjectsSummary, row)
		projectSummarySpinnerSuccess.Success(repoWithOwner + " MetaData fetched successfully")
	}

	return gitlabProjectsSummary
}
