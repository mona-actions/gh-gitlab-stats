package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/internal/api"
	"github.com/mona-actions/gh-gitlab-stats/internal/models"
	"github.com/mona-actions/gh-gitlab-stats/internal/services"
	"github.com/mona-actions/gh-gitlab-stats/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

var (
	debug     bool
	hostname  string
	input     string
	namespace string
	output    string
	repoList  string
	token     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitlab-stats",
	Short: "GitLab Repository Statistics Tool",
	Long: `A GitHub CLI extension for scanning GitLab instances and generating 
repository statistics reports similar to GitHub's repository inventory.

This tool connects to GitLab instances (including GitLab.com or self-hosted)
and generates comprehensive statistics about repositories, including:
- Repository metadata and settings
- Collaboration statistics (issues, merge requests, members)
- Activity metrics (commits, releases, tags)
- Storage and size information

The output is compatible with GitHub repository analysis tools.`,
	RunE: runGLRepoStats,
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
	cobra.OnInitialize(initConfig)

	// Command flags matching the specification
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging with detailed progress output")
	rootCmd.Flags().StringVarP(&hostname, "hostname", "H", "gitlab.com", "GitLab hostname (without https:// prefix)")
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "Path to file with list of namespaces to scan (one per line)")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "GitLab namespace/group to analyze (e.g., \"mygroup/subgroup\")")
	rootCmd.Flags().StringVarP(&output, "output", "O", "csv", "Output format: \"csv\" (timestamped file) or \"table\" (console)")
	rootCmd.Flags().StringVarP(&repoList, "repo-list", "r", "", "Path to file with list of repositories in \"namespace/project\" format (one per line)")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "GitLab Personal Access Token (required, or set GITLAB_TOKEN env var)")

	// Bind flags to viper
	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	viper.BindPFlag("hostname", rootCmd.Flags().Lookup("hostname"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("namespace", rootCmd.Flags().Lookup("namespace"))
	viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))

	// Set environment variable prefix and bindings
	viper.SetEnvPrefix("GITLAB")
	viper.BindEnv("token") // Binds to GITLAB_TOKEN
} // initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gh-gitlab-stats" (without extension).
		viper.AddConfigPath(home + "/.gh-gitlab-stats")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && debug {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// runGLRepoStats is the main function that executes the GitLab repository statistics collection
func runGLRepoStats(cmd *cobra.Command, args []string) error {
	// Get token from viper (supports flag, env var, or config file)
	if token == "" {
		token = viper.GetString("token")
	}

	// Get other values from viper if not set via flags
	if hostname == "" || hostname == "gitlab.com" {
		if viperHostname := viper.GetString("hostname"); viperHostname != "" {
			hostname = viperHostname
		}
	}

	// Normalize output format to lowercase for consistent internal use
	output = strings.ToLower(output)

	if debug {
		fmt.Println("Debug mode enabled")
	}

	// Validate inputs
	if err := validateInputs(); err != nil {
		return err
	}

	// Setup client and scanner
	gitlabURL := buildGitLabURL()
	client, err := api.NewRestClient(gitlabURL, token)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	scanner := services.NewScanner(client)

	// Run scan
	fmt.Printf("Starting GitLab repository statistics collection...\n")
	allStats, err := executeScan(cmd.Context(), client, scanner, gitlabURL, output, debug)
	if err != nil {
		return err
	}

	// Write output
	return writeOutput(allStats)
}

// validateInputs validates command-line flags
func validateInputs() error {
	if token == "" {
		return fmt.Errorf("GitLab token is required. Use --token flag or set GITLAB_TOKEN environment variable")
	}
	if output != "csv" && output != "table" {
		return fmt.Errorf("invalid output format: %s. Must be 'csv' or 'table'", output)
	}
	return nil
}

// buildGitLabURL constructs the GitLab URL from hostname
func buildGitLabURL() string {
	if strings.HasPrefix(hostname, "http://") || strings.HasPrefix(hostname, "https://") {
		return hostname
	}
	return "https://" + hostname
}

// readLinesFromFile reads non-empty, non-comment lines from a file
func readLinesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// executeScan performs the repository scan based on input parameters
func executeScan(ctx context.Context, client *api.RestClient, scanner *services.Scanner, gitlabURL, outputFormat string, verbose bool) ([]*models.RepositoryStats, error) {
	progressReporter := createProgressReporter()

	// Handle specific repository list
	if repoList != "" {
		return scanSpecificRepositories(ctx, client)
	}

	// Handle namespaces
	namespaces, err := getNamespacesToScan()
	if err != nil {
		return nil, err
	}

	// Scan with server-side filtering for namespaces
	if len(namespaces) == 0 {
		// No namespace filter - scan all accessible projects
		scanOptions := &models.ScanOptions{
			GitLabURL:    gitlabURL,
			Token:        token,
			OutputFormat: outputFormat,
			Verbose:      verbose,
			MaxProjects:  0,
		}
		result, err := scanner.ScanRepositories(ctx, scanOptions, progressReporter)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		return result.RepositoryStats, nil
	}

	// Scan specific namespaces with server-side filtering
	return scanNamespaces(ctx, scanner, gitlabURL, outputFormat, verbose, progressReporter, namespaces)
}

// createProgressReporter creates the appropriate progress reporter
func createProgressReporter() ui.ProgressReporter {
	if output == "table" || debug {
		return ui.NewConsoleProgress()
	}
	return ui.NewQuietProgress()
}

// getNamespacesToScan returns the list of namespaces to scan
func getNamespacesToScan() ([]string, error) {
	if input != "" {
		namespaces, err := readLinesFromFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read namespaces from file %s: %w", input, err)
		}
		if debug {
			fmt.Printf("Read %d namespaces from file: %s\n", len(namespaces), input)
		}
		return namespaces, nil
	}

	if namespace != "" {
		if debug {
			fmt.Printf("Scanning namespace: %s\n", namespace)
		}
		return []string{namespace}, nil
	}

	return nil, nil
}

// scanSpecificRepositories scans a list of specific repositories
func scanSpecificRepositories(ctx context.Context, client *api.RestClient) ([]*models.RepositoryStats, error) {
	repositories, err := readLinesFromFile(repoList)
	if err != nil {
		return nil, fmt.Errorf("failed to read repositories from file %s: %w", repoList, err)
	}

	if debug {
		fmt.Printf("Read %d repositories from file: %s\n", len(repositories), repoList)
	}

	var allStats []*models.RepositoryStats
	for _, repoPath := range repositories {
		if debug {
			fmt.Printf("Scanning repository: %s\n", repoPath)
		}

		if strings.Count(repoPath, "/") < 1 {
			fmt.Printf("Warning: Invalid repository path format: %s (expected namespace/project)\n", repoPath)
			continue
		}

		project, err := client.GetProject(ctx, repoPath)
		if err != nil {
			fmt.Printf("Warning: Failed to get project %s: %v\n", repoPath, err)
			continue
		}

		stats, err := client.GetProjectStatistics(ctx, project.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to get statistics for %s: %v\n", repoPath, err)
			continue
		}

		repoStats := services.ConvertToRepoStats(project, stats)
		allStats = append(allStats, repoStats)
	}

	return allStats, nil
}

// scanNamespaces scans specific namespaces and returns results
// Uses server-side filtering for efficiency - no client-side filtering needed
func scanNamespaces(ctx context.Context, scanner *services.Scanner, gitlabURL, outputFormat string, verbose bool, progressReporter ui.ProgressReporter, namespaces []string) ([]*models.RepositoryStats, error) {
	var allStats []*models.RepositoryStats

	for _, ns := range namespaces {
		if verbose {
			fmt.Printf("Processing namespace: %s (using server-side filtering)\n", ns)
		}

		// Create scan options with namespace for server-side filtering
		scanOptions := &models.ScanOptions{
			GitLabURL:    gitlabURL,
			Token:        token,
			Namespace:    ns, // Server-side filtering by namespace
			OutputFormat: outputFormat,
			Verbose:      verbose,
			MaxProjects:  0,
		}

		result, err := scanner.ScanRepositories(ctx, scanOptions, progressReporter)
		if err != nil {
			return nil, fmt.Errorf("scan failed for namespace %s: %w", ns, err)
		}

		// No client-side filtering needed - server already filtered by namespace
		allStats = append(allStats, result.RepositoryStats...)
	}

	return allStats, nil
}

// writeOutput writes the scan results to the appropriate output format
func writeOutput(allStats []*models.RepositoryStats) error {
	if output == "table" {
		return outputTable(allStats)
	}

	// CSV output
	outputFile := fmt.Sprintf("gitlab-stats-%s.csv", time.Now().Format("2006-01-02-15-04-05"))
	formatter, err := ui.NewFormatter("csv")
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	if err := formatter.WriteToFile(allStats, outputFile); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("\nScan completed successfully!\n")
	fmt.Printf("Total repositories processed: %d\n", len(allStats))
	fmt.Printf("Output written to: %s\n", outputFile)
	return nil
}

// outputTable outputs the results in table format
func outputTable(stats []*models.RepositoryStats) error {
	if len(stats) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	// Table header
	fmt.Printf("%-30s %-30s %-10s %-15s %-15s %-10s %-10s %-15s %-10s %-10s\n",
		"Namespace", "Repository", "Empty", "Size(MB)", "LFS(MB)", "Commits", "Issues", "MRs", "Branches", "Tags")
	fmt.Println(strings.Repeat("-", 155))

	// Table rows
	for _, stat := range stats {
		fmt.Printf("%-30s %-30s %-10v %-15.2f %-15.2f %-10d %-10d %-15d %-10d %-10d\n",
			truncate(stat.Namespace, 30),
			truncate(stat.RepoName, 30),
			stat.IsEmpty,
			stat.RepoSizeMB,
			stat.LFSSizeMB,
			stat.CommitCount,
			stat.IssueCount,
			stat.MRCount,
			stat.BranchCount,
			stat.TagCount)
	}

	fmt.Printf("\nTotal repositories: %d\n", len(stats))
	return nil
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
