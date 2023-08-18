/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"time"

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
	IssueEventCount         int
	ReleaseCount            int
	BranchCount             int
	TagCount                int
	DiscussionCount         int
	HasWiki                 bool
	FullUrl                 string
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

	rootCmd.Flags().StringP("gitlab-hostname", "s", "", "The hostname of the GitLab instance to gather metrics from E.g https://gitlab.company.com")
	rootCmd.MarkFlagRequired("gitlab-hostname")

	rootCmd.Flags().StringP("token", "t", "", "The token to use to authenticate to the GitLab instance")
	rootCmd.MarkFlagRequired("token")
}

func getGitlabStats(cmd *cobra.Command, args []string) {
	gitlabHostname := cmd.Flag("gitlab-hostname").Value.String()
	gitlabToken := cmd.Flag("token").Value.String()
	client := initClient(gitlabHostname, gitlabToken)
	//getNamespaces(client)
	groupSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Groups")
	getGroups(client)
	groupSpinnerSuccess.Success("Groups fetched successfully")

	projectSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Projects")
	gitlabProjects := getProjects(client)
	projectSpinnerSuccess.Success("Projects fetched successfully")

	for _, project := range gitlabProjects {
		log.Println("Found project", project.Name)
		commitSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Commits")
		commits := getCommitActivity(project, client)
		commitSpinnerSuccess.Success("Commits fetched successfully")

		mergeRequestSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Merge Requests")
		mergeRequests := getMergeRequests(project, client)
		mergeRequestSpinnerSuccess.Success("Merge Requests fetched successfully")

		row := &ProjectSummary{
			Namespace:   project.Namespace.Name,
			ProjectName: project.Name,
			IsEmpty:     project.EmptyRepo,
			Last_Update: project.LastActivityAt,
			RepoSize:    project.Statistics.RepositorySize,
			IsFork:      project.ForkedFromProject != nil,
			//CollaboratorCount:
			//ProtectedBranchCount:
			//MergeRequestReviewCount:
			//MilestoneCount:
			//IssueCount:
			MergeRequestCount: len(mergeRequests),
			//MRReviewCommentCount:
			CommitCommentCount: len(commits),
			//IssueCommentCount:
			//IssueEventCount:
			//ReleaseCount:
			//BranchCount:
			TagCount: len(project.TagList),
			//DiscussionCount:
			HasWiki: project.WikiEnabled,
			FullUrl: project.WebURL,
		}
		gitlabProjectsSummary = append(gitlabProjectsSummary, row)
	}

	projectsSummary = convertToCSVFormat(gitlabProjectsSummary)
	createCSV(projectsSummary, "projects.csv")
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

func getProjects(client *gitlab.Client) []*gitlab.Project {

	var projects []*gitlab.Project
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Statistics: gitlab.Bool(true),
	}

	for {
		p, response, err := client.Projects.ListProjects(opt)

		if err != nil {
			log.Fatalf("Failed to list projects: %v", err)
		}
		projects = append(projects, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return projects
}

func getCommitActivity(project *gitlab.Project, client *gitlab.Client) []*gitlab.Commit {
	var commits []*gitlab.Commit
	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	for {
		c, response, err := client.Commits.ListCommits(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list commits: %v %v", response, err)
		}
		commits = append(commits, c...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	//TODO: Need to decide if we would like to build a new struct for more readable Commit Summary or just use the gitlab.Commit struct
	for _, commit := range commits {
		log.Println("Found commit", commit.ID, commit.ProjectID, commit.Title)
	}
	return commits
}

func getMergeRequests(project *gitlab.Project, client *gitlab.Client) []*gitlab.MergeRequest {
	var mergeRequests []*gitlab.MergeRequest
	opt := &gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.MergeRequests.ListProjectMergeRequests(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list merge requests: %v %v", response, err)
		}
		mergeRequests = append(mergeRequests, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, mergeRequest := range mergeRequests {
		log.Println("Found merge request: ", mergeRequest.ID, mergeRequest.Title, mergeRequest.Author.Username)
	}

	return mergeRequests
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
	header := []string{"Namespace Name", "Name", "Empty Repo", "Last Activity At", "Repository Size", "Commit Count"}
	rows = append(rows, header)

	// Add project rows
	for _, project := range projects {
		row := []string{
			project.Namespace,
			project.ProjectName,
			strconv.FormatBool(project.IsEmpty),
			project.Last_Update.Format(time.RFC3339),
			strconv.FormatInt(project.RepoSize, 10),
			strconv.Itoa(project.CommitCommentCount),
			strconv.Itoa(project.MergeRequestCount),
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
