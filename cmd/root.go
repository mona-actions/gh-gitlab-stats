/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"
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
var rootCmd = &cobra.Command{
	Use:   "gh gitlab-stats",
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

	rootCmd.Flags().StringP("hostname", "s", "", "The hostname of the GitLab instance to gather metrics from E.g https://gitlab.company.com")
	rootCmd.MarkFlagRequired("hostname")

	rootCmd.Flags().StringP("token", "t", "", "The token to use to authenticate to the GitLab instance")
	rootCmd.MarkFlagRequired("token")

	rootCmd.Flags().StringP("output-file", "f", "gitlab-stats-"+timestamp+".csv", "The output file name to write the results to")
}

func getGitlabStats(cmd *cobra.Command, args []string) {

	gitlabHostname := cmd.Flag("hostname").Value.String()
	gitlabToken := cmd.Flag("token").Value.String()
	outputFileName := cmd.Flag("output-file").Value.String()
	client := initClient(gitlabHostname, gitlabToken)
	//getNamespaces(client)
	groupSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Groups")
	groups.GetGroups(client)
	groupSpinnerSuccess.Success("Groups fetched successfully")

	projectSpinnerSuccess, _ := pterm.DefaultSpinner.Start("Fetching Projects")
	gitlabProjects := projects.GetProjects(client)
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
