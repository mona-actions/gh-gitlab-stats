/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/api/commits"
	"github.com/mona-actions/gh-gitlab-stats/api/issues"
	"github.com/mona-actions/gh-gitlab-stats/api/members"
	"github.com/mona-actions/gh-gitlab-stats/api/mergerequests"
	"github.com/mona-actions/gh-gitlab-stats/api/projects"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type GitLabSummary []ProjectSummary

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
	projectsSummary       [][]string
	gitlabProjectsSummary []*ProjectSummary
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-gitlab-stats",
	Short: "gh cli extension for analyzing GitLab Instance",
	Long: `gh cli extension for analyzing GitLab Instance to get migration statistics of
	      repositories, issues...`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: getGitlabStats,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-gitlab-stats.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	timestamp := time.Now().Format("2006-01-02-15-04-05")

	rootCmd.Flags().StringP("gitlab-hostname", "s", "", "The hostname of the GitLab instance to gather metrics from E.g https://gitlab.company.com")
	rootCmd.MarkFlagRequired("gitlab-hostname")

	rootCmd.Flags().StringP("token", "t", "", "The token to use to authenticate to the GitLab instance")
	rootCmd.MarkFlagRequired("token")

	rootCmd.Flags().StringP("output-file", "f", "gitlab-stats-"+timestamp+".csv", "The output file name to write the results to")
}

func getGitlabStats(cmd *cobra.Command, args []string) {

	gitlabHostname := cmd.Flag("gitlab-hostname").Value.String()
	gitlabToken := cmd.Flag("token").Value.String()
	outputFileName := cmd.Flag("output-file").Value.String()
	client := initClient(gitlabHostname, gitlabToken)
	//getNamespaces(client)
	groupSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Groups")
	getGroups(client)
	groupSpinnerSuccess.Success("Groups fetched successfully")

	projectSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Projects")
	gitlabProjects := projects.GetProjects(client)
	projectSpinnerSuccess.Success("Projects fetched successfully")

	gitlabProjectsSummary := getProjectSummary(gitlabProjects, client)

	projectsSummary = convertToCSVFormat(gitlabProjectsSummary)
	createCSV(projectsSummary, outputFileName)
}

func initClient(hostname string, token string) *gitlab.Client {
	var git *gitlab.Client
	var err error
	if hostname == "" {
		git, err = gitlab.NewClient(token)
	} else {
		git, err = gitlab.NewClient(token, gitlab.WithBaseURL(hostname))
	}

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return git
}

func getProjectSummary(gitlabProjects []*gitlab.Project, client *gitlab.Client) []*ProjectSummary {

	isMigrationIssue := false
	var issueCommentCount int
	var mergeRequestCommentCount int

	for _, project := range gitlabProjects {
		log.Println("Found project", project.Name)
		commitSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Commits")
		commits := commits.GetCommitActivity(project, client)
		commitSpinnerSuccess.Success("Commits fetched successfully")

		mergeRequestSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Merge Requests")
		mergeRequests := mergerequests.GetMergeRequests(project, client)

		for _, mergeRequest := range mergeRequests {
			mergeRequestComments := mergerequests.GetMergeRequestComments(project, mergeRequest, client)
			mergeRequestCommentCount += len(mergeRequestComments)
		}
		mergeRequestSpinnerSuccess.Success("Merge Requests fetched successfully")

		projectMembersSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Project Members")
		projectMembers := members.GetProjectMembers(project, client)
		projectMembersSpinnerSuccess.Success("Project Members fetched successfully")

		projectBranchesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Project Branches")
		projectBranches := projects.GetProjectBranches(project, client)
		projectBranchesSpinnerSuccess.Success("Project Branches fetched successfully")

		projectMilestonesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Project Milestones")
		projectMilestones := projects.GetProjectMilestones(project, client)
		projectMilestonesSpinnerSuccess.Success("Project Milestones fetched successfully")

		projectIssuesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Project Issues")
		projectIssues := issues.GetProjectIssues(project, client)

		for _, issue := range projectIssues {
			issueComments := issues.GetIssueComments(project, issue, client)
			issueCommentCount += len(issueComments)
		}
		fmt.Println("No. issue comments: ", issueCommentCount)
		projectIssuesSpinnerSuccess.Success("Project Issues fetched successfully")

		projectReleasesSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Project Releases")
		projectReleases := projects.GetProjectReleases(project, client)
		projectReleasesSpinnerSuccess.Success("Project Releases fetched successfully")

		recordCount := len(commits) + len(projectIssues) + len(mergeRequests) + len(projectMilestones) + len(projectReleases) + len(projectBranches) + len(project.TagList) + mergeRequestCommentCount + issueCommentCount
		repoSizeInMB := (project.Statistics.RepositorySize / 1000000)
		if recordCount > 60000 || repoSizeInMB > 1500 {
			isMigrationIssue = true
		}
		row := &ProjectSummary{
			Namespace:         project.Namespace.Name,
			ProjectName:       project.Name,
			IsEmpty:           project.EmptyRepo,
			Last_Update:       project.LastActivityAt,
			IsFork:            project.ForkedFromProject != nil,
			RepoSize:          repoSizeInMB,
			RecordCount:       recordCount,
			CollaboratorCount: len(projectMembers),
			//ProtectedBranchCount: len(projectBranches),
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
			HasWiki:        project.WikiEnabled,
			FullUrl:        project.WebURL,
			MigrationIssue: isMigrationIssue,
		}
		gitlabProjectsSummary = append(gitlabProjectsSummary, row)
	}

	return gitlabProjectsSummary
}

func getGroups(client *gitlab.Client) []*gitlab.Group {
	var groups []*gitlab.Group
	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	//TODO: Check to see if pagination can be extrapolated to a function
	for {
		g, response, err := client.Groups.ListGroups(opt)

		if err != nil {
			log.Fatalf("Failed to list groups: %v", err)
		}
		groups = append(groups, g...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, group := range groups {
		log.Println("Found group", group.Name)
	}

	return groups
}

func createCSV(data [][]string, filename string) {
	// Create team membership csv
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Initialize csv writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write team memberships to csv

	for _, line := range data {
		writer.Write(line)
	}
}

func convertToCSVFormat(projects []*ProjectSummary) [][]string {
	var rows [][]string

	// Add header row
	header := []string{"Namespace Name", "Project_Name", "Is_Empty", "Last_Push", "Last_Update", "isFork", "Repository_Size(mb)", "Record_Count", "Collaborator_Count", "Protected_Branch_Count", "MR_Review_Count", "Milestone_Count", "Issue_Count", "MergeRequest_Count", "MR_Review_Comment_Count", "Commit_Comment_Count", "Issue_Comment_Count", "Issue_Event_Count", "Release_Count", "Project_Count", "Branch_Count", "Tag_Count", "Discussion_Count", "Has Wiki", "Full_URL", "Migration_Issue"}
	rows = append(rows, header)

	// Add project rows
	for _, project := range projects {
		row := []string{
			project.Namespace,
			project.ProjectName,
			strconv.FormatBool(project.IsEmpty),
			"N\\A",
			project.Last_Update.Format(time.RFC3339),
			strconv.FormatBool(project.IsFork),
			strconv.FormatInt(project.RepoSize, 10),
			strconv.Itoa(project.RecordCount),
			strconv.Itoa(project.CollaboratorCount),
			"Protected Branch Count To be implemented",
			" Mr Review Count To be implemented",
			strconv.Itoa(project.MilestoneCount),
			strconv.Itoa(project.IssueCount),
			strconv.Itoa(project.MergeRequestCount),
			strconv.Itoa(project.MRReviewCommentCount),
			strconv.Itoa(project.CommitCommentCount),
			strconv.Itoa(project.IssueCommentCount),
			"N\\A",
			strconv.Itoa(project.ReleaseCount),
			"N\\A",
			strconv.Itoa(project.BranchCount),
			strconv.Itoa(project.TagCount),
			"N\\A",
			strconv.FormatBool(project.HasWiki),
			project.FullUrl,
		}
		rows = append(rows, row)
	}

	return rows
}

// Namespace will return all users and groups together
// func getNamespaces(client *gitlab.Client) []*gitlab.Namespace {
// 	namespaces, _, err := client.Namespaces.ListNamespaces(&gitlab.ListNamespacesOptions{})

// 	if err != nil {
// 		log.Fatalf("Failed to list projects: %v", err)
// 	}

// 	for _, namespaces := range namespaces {
// 		log.Println("Found namespace", namespaces.Name)
// 	}
// 	return namespaces
// }
