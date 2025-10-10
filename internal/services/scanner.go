package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/internal/api"
	"github.com/mona-actions/gh-gitlab-stats/internal/models"
	"github.com/mona-actions/gh-gitlab-stats/internal/ui"
)

// Scanner handles the scanning of GitLab repositories
type Scanner struct {
	client api.GitLabClient
}

// NewScanner creates a new scanner instance
func NewScanner(client api.GitLabClient) *Scanner {
	return &Scanner{
		client: client,
	}
}

// ScanRepositories scans GitLab repositories and collects statistics
func (s *Scanner) ScanRepositories(ctx context.Context, options *models.ScanOptions, progress ui.ProgressReporter) (*models.ScanResult, error) {
	start := time.Now()

	result := &models.ScanResult{
		RepositoryStats: []*models.RepositoryStats{},
		Errors:          []error{},
	}

	// Get list of projects
	fmt.Println("\n🔍 Discovering projects...")
	projects, err := s.getProjects(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	result.TotalProjects = len(projects)

	fmt.Printf("✓ Found %d projects to scan\n", len(projects))
	if options.Verbose {
		fmt.Printf("  Using %d parallel workers for scanning\n", func() int {
			numWorkers := 5
			if options.MaxProjects > 0 && options.MaxProjects < numWorkers {
				numWorkers = options.MaxProjects
			}
			return numWorkers
		}())
	}
	fmt.Println()

	// Initialize progress
	progress.Start(len(projects))

	// Create channels for worker communication
	projectChan := make(chan *api.Project, len(projects))
	resultChan := make(chan *models.RepositoryStats, len(projects))
	errorChan := make(chan error, len(projects))

	// Start workers
	var wg sync.WaitGroup
	numWorkers := 5 // Default parallel workers
	if options.MaxProjects > 0 && options.MaxProjects < numWorkers {
		numWorkers = options.MaxProjects
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.worker(ctx, projectChan, resultChan, errorChan, &wg, options.Verbose)
	}

	// Send projects to workers
	go func() {
		defer close(projectChan)
		for _, project := range projects {
			select {
			case projectChan <- project:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Process results
	for {
		select {
		case stat, ok := <-resultChan:
			if !ok {
				goto done
			}
			if stat != nil {
				result.RepositoryStats = append(result.RepositoryStats, stat)
				result.ProcessedProjects++
				logProgress(options.Verbose, result.ProcessedProjects, result.TotalProjects, stat)
				progress.Update(result.ProcessedProjects)
			}
		case err, ok := <-errorChan:
			if !ok {
				continue
			}
			if err != nil {
				result.Errors = append(result.Errors, err)
				if options.Verbose {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
			}
		case <-ctx.Done():
			progress.Finish()
			return nil, ctx.Err()
		}
	}

done:
	progress.Finish()
	result.Duration = time.Since(start)
	printScanSummary(result)
	return result, nil
}

// getProjects retrieves the list of projects to scan
func (s *Scanner) getProjects(ctx context.Context, options *models.ScanOptions) ([]*api.Project, error) {
	// When no GroupID is specified, we want to get ALL accessible projects
	// By default (without Membership parameter), GitLab returns all projects visible to the user

	trueVal := true
	listOptions := &api.ListProjectsOptions{
		Page:       1,
		PerPage:    100,
		Statistics: &trueVal,
		Archived:   nil, // Get ALL projects (both archived and non-archived)
	}

	if options.GroupID != nil {
		listOptions.GroupID = options.GroupID
	}

	var allProjects []*api.Project

	for {
		projects, err := s.client.ListProjects(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to list projects (page %d): %w", listOptions.Page, err)
		}

		if options.Verbose {
			fmt.Printf("DEBUG: Page %d returned %d projects\n", listOptions.Page, len(projects))
		}

		if len(projects) == 0 {
			break
		}

		allProjects = append(allProjects, projects...)

		// Check if we've hit the max projects limit
		if options.MaxProjects > 0 && len(allProjects) >= options.MaxProjects {
			allProjects = allProjects[:options.MaxProjects]
			break
		}

		// If we got fewer projects than requested per page, we're done
		if len(projects) < listOptions.PerPage {
			break
		}

		// Move to next page
		listOptions.Page++
	}

	if options.Verbose {
		fmt.Printf("DEBUG: Total projects found across all pages: %d\n", len(allProjects))
	}

	return allProjects, nil
}

// worker processes individual projects
func (s *Scanner) worker(ctx context.Context, projectChan <-chan *api.Project, resultChan chan<- *models.RepositoryStats, errorChan chan<- error, wg *sync.WaitGroup, verbose bool) {
	defer wg.Done()

	for project := range projectChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stat, err := s.processProject(ctx, project, verbose)
		if err != nil {
			errorChan <- fmt.Errorf("error processing project %s: %w", project.PathWithNamespace, err)
			continue
		}

		resultChan <- stat
	}
}

// processProject collects comprehensive statistics for a single project
func (s *Scanner) processProject(ctx context.Context, project *api.Project, verbose bool) (*models.RepositoryStats, error) {
	if verbose {
		fmt.Printf("  → Processing: %s (ID: %d)\n", project.PathWithNamespace, project.ID)
		fmt.Printf("    Fetching detailed statistics...\n")
	}

	// Get detailed statistics
	stats, err := s.client.GetProjectStatistics(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project statistics: %w", err)
	}

	if verbose {
		fmt.Printf("    ✓ Retrieved: branches(%d), tags(%d), members(%d), issues(%d), MRs(%d)\n",
			stats.BranchCount, stats.TagCount, stats.MemberCount, stats.IssueCount, stats.MergeRequestCount)
		fmt.Printf("    ✓ Reviews: MR Reviews(%d) | Commits(%d)\n",
			stats.MergeRequestReviewCount, stats.CommitCount)
		fmt.Printf("    ✓ Comments: MR(%d), Issue(%d)\n",
			stats.MergeRequestCommentCount, stats.IssueCommentCount)
	}

	return ConvertToRepoStats(project, stats), nil
}

// ConvertToRepoStats converts API project and statistics to repository stats model
func ConvertToRepoStats(project *api.Project, stats *api.ProjectStatistics) *models.RepositoryStats {
	return &models.RepositoryStats{
		Namespace:            extractNamespace(project.PathWithNamespace),
		RepoName:             project.Name,
		IsEmpty:              project.EmptyRepo,
		LastPush:             project.LastActivityAt,
		LastUpdate:           project.LastActivityAt,
		IsFork:               project.ForkedFromProject,
		IsArchive:            project.Archived,
		RepoSizeMB:           float64(stats.RepositorySize) / (1024 * 1024),
		LFSSizeMB:            float64(stats.LFSObjectsSize) / (1024 * 1024),
		RecordCount:          stats.CommitCount,
		CollaboratorCount:    stats.MemberCount,
		ProtectedBranchCount: countProtectedBranches(stats.BranchCount),
		PRReviewCount:        stats.MergeRequestReviewCount,
		MilestoneCount:       stats.MilestoneCount,
		IssueCount:           stats.IssueCount,
		PRCount:              stats.MergeRequestCount,
		PRReviewCommentCount: stats.MergeRequestCommentCount,
		CommitCount:          stats.CommitCount,
		IssueCommentCount:    stats.IssueCommentCount,
		ReleaseCount:         stats.ReleaseCount,
		BranchCount:          stats.BranchCount,
		TagCount:             stats.TagCount,
		HasWiki:              stats.HasWikiPages,
		FullURL:              project.WebURL,
		Created:              project.CreatedAt,
	}
}

// extractNamespace extracts the full namespace path from the path with namespace
// For "group/subgroup/project", returns "group/subgroup"
// For "user/project", returns "user"
func extractNamespace(pathWithNamespace string) string {
	lastSlash := strings.LastIndex(pathWithNamespace, "/")
	if lastSlash > 0 {
		return pathWithNamespace[:lastSlash]
	}
	return ""
}

// countProtectedBranches estimates protected branches (GitLab doesn't provide this directly)
// Assumes main/master branch is protected + ~10% of other branches
func countProtectedBranches(totalBranches int) int {
	if totalBranches == 0 {
		return 0
	}
	if totalBranches == 1 {
		return 1
	}
	protected := 1 + (totalBranches-1)/10
	if protected > totalBranches {
		return totalBranches
	}
	return protected
}

// logProgress outputs progress information based on verbosity level
func logProgress(verbose bool, current, total int, stat *models.RepositoryStats) {
	if verbose {
		fmt.Printf("\n[%d/%d] ✓ Scanned: %s/%s\n", current, total, stat.Namespace, stat.RepoName)
		fmt.Printf("    Size: %.0f MB | LFS: %.0f MB | Commits: %d | Issues: %d | MRs: %d | Branches: %d | Tags: %d\n",
			stat.RepoSizeMB, stat.LFSSizeMB, stat.RecordCount, stat.IssueCount, stat.PRCount, stat.BranchCount, stat.TagCount)
	} else {
		fmt.Printf("\r[%d/%d] Scanning projects... Current: %s/%s",
			current, total, truncate(stat.Namespace, 20), truncate(stat.RepoName, 30))
	}
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

// printScanSummary prints the final scan summary
func printScanSummary(result *models.ScanResult) {
	avgTime := time.Duration(0)
	if result.ProcessedProjects > 0 {
		avgTime = result.Duration / time.Duration(result.ProcessedProjects)
	}

	fmt.Printf("\n\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("                    SCAN COMPLETE\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("  Total projects found:     %d\n", result.TotalProjects)
	fmt.Printf("  Successfully processed:   %d\n", result.ProcessedProjects)
	fmt.Printf("  Errors encountered:       %d\n", len(result.Errors))
	fmt.Printf("  Duration:                 %v\n", result.Duration.Round(time.Second))
	fmt.Printf("  Average time per project: %v\n", avgTime.Round(time.Millisecond))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")
}
