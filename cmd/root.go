/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/api/groups"
	"github.com/mona-actions/gh-gitlab-stats/api/projects"
	"github.com/mona-actions/gh-gitlab-stats/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var (
	projectsSummary [][]string
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "gh gitlab-stats",
		Short: "gh cli extension for analyzing GitLab Instance",
		Long: `gh cli extension for analyzing GitLab Instance to get migration statistics of
	      repositories, issues...`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: getGitlabStats,
	}
)

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

	rootCmd.Flags().StringP("hostname", "s", "", "The hostname/server of the GitLab instance to gather metrics from E.g https://gitlab.company.com")
	rootCmd.MarkFlagRequired("hostname")

	rootCmd.Flags().StringP("token", "t", "", "The token to use to authenticate to the GitLab instance")
	rootCmd.MarkFlagRequired("token")

	rootCmd.Flags().StringP("output-file", "f", "gitlab-stats-"+timestamp+".csv", "The output file name to write the results to")

	rootCmd.Flags().StringP("groups", "g", "", "The specific groups to gather metrics from. E.g group1,group2,group3")
}

func getGitlabStats(cmd *cobra.Command, args []string) {
	// Init Variables
	gitlabHostname := cmd.Flag("hostname").Value.String()
	groupNames := cmd.Flag("groups").Value.String()
	gitlabToken := cmd.Flag("token").Value.String()
	outputFileName := cmd.Flag("output-file").Value.String()
	var gitlabGroups []*gitlab.Group
	var gitlabProjects []*gitlab.Project
	checkVars(cmd)
	if !strings.HasPrefix(gitlabHostname, "http://") && !strings.HasPrefix(gitlabHostname, "https://") {
		gitlabHostname = "https://" + gitlabHostname
	}

	//Init GitLab Client
	client := initClient(gitlabHostname, gitlabToken)

	if groupNames != "" {
		groupSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Groups")
		gitlabGroups = internal.GetGroupsFromNames(client, groupNames)
		if len(gitlabGroups) == 0 {
			groupSpinnerSuccess.Info("No groups found")
			os.Exit(0)
		}
		groupSpinnerSuccess.Success("Groups fetched successfully")
	}

	projectSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Projects")
	if groupNames != "" {
		gitlabProjects = GetGitLabGroupsProjects(client, gitlabGroups)
	} else {
		gitlabProjects = projects.GetProjects(client)
	}
	projectSpinnerSuccess.Success("Projects fetched successfully")

	gitlabProjectsSummary := internal.GetProjectSummary(gitlabProjects, client)

	csvFileSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Creating CSV File")
	projectsSummary = internal.ConvertToCSVFormat(gitlabProjectsSummary)
	internal.CreateCSV(projectsSummary, outputFileName)
	csvFileSpinnerSuccess.Success("CSV File created successfully")
}

func initClient(hostname string, token string) *gitlab.Client {
	var git *gitlab.Client
	var err error
	git, err = gitlab.NewClient(token, gitlab.WithBaseURL(hostname))
	gitlabClientSpinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Authenticating to GitLab Host: %s", hostname))
	if err != nil {
		gitlabClientSpinner.Fail("Failed to create GitLab Client")
		log.Fatalf("Failed to create client: %+v", err)
	}
	_, _, err = git.Users.CurrentUser()
	if err != nil {
		gitlabClientSpinner.Fail("Failed to authenticate to GitLab")
		log.Fatalf("Failed to authenticate: %+v", err)
	}
	gitlabClientSpinner.Success("Authenticated to GitLab")
	return git
}

func checkVars(cmd *cobra.Command) {
	gitlabHostname := cmd.Flag("hostname").Value.String()
	// Check if the hostname is "gitlab.com" or "https://gitlab.com"
	if gitlabHostname == "gitlab.com" || gitlabHostname == "https://gitlab.com" || gitlabHostname == "http://gitlab.com" {
		log.Fatalf("The hostname cannot be gitlab.com")
	} else if gitlabHostname == "" {
		log.Fatalf("The hostname cannot be empty")
	}
}

func GetGitLabGroupsProjects(client *gitlab.Client, gitlabGroups []*gitlab.Group) []*gitlab.Project {
	var gitlabProjects []*gitlab.Project

	// Get all projects in the specified groups
	groupsProjects := groups.GetGroupsProjects(client, gitlabGroups)
	for _, project := range groupsProjects {

		// Get the project details with statistics
		gitlabProject := projects.GetProject(project, client)
		gitlabProjects = append(gitlabProjects, gitlabProject)
	}
	return gitlabProjects
}
