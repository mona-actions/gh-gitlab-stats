/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gh-gitlab-stats",
	Short: "gh cli extension for analyzing GitLab Instance",
	Long: `gh cli extension for analyzing GitLab Instance to get migration statistics of
	      repositories, issues...`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		gitlabHostname := cmd.Flag("gitlab-hostname").Value.String()
		gitlabToken := cmd.Flag("token").Value.String()
		initClient(gitlabHostname, gitlabToken)
	},
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

func initClient(hostname string, token string) {
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
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{})
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}

	for _, user := range users {
		log.Println("Found user", user.Name)
	}
}
