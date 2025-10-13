package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/mona-actions/gh-gitlab-stats/internal/models"
)

// Formatter interface for different output formats
type Formatter interface {
	WriteToFile(stats []*models.RepositoryStats, filename string) error
}

// NewFormatter creates a formatter based on the format type
func NewFormatter(format string) (Formatter, error) {
	switch format {
	case "csv":
		return &CSVFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (only 'csv' is supported)", format)
	}
}

// CSVFormatter formats output as CSV
type CSVFormatter struct{}

// WriteToFile writes repository statistics to a CSV file
func (f *CSVFormatter) WriteToFile(stats []*models.RepositoryStats, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if len(stats) == 0 {
		return nil
	}

	// Write header
	header := getCSVHeaders()
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data rows
	for _, stat := range stats {
		row := convertToCSVRow(stat)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// getCSVHeaders returns the CSV header row
func getCSVHeaders() []string {
	return []string{
		"Namespace",
		"Project",
		"Is_Empty",
		"isFork",
		"isArchive",
		"Project_Size(mb)",
		"LFS_Size(mb)",
		"Collaborator_Count",
		"Protected_Branch_Count",
		"MR_Review_Count",
		"Milestone_Count",
		"Issue_Count",
		"MR_Count",
		"MR_Review_Comment_Count",
		"Commit_Count",
		"Issue_Comment_Count",
		"Release_Count",
		"Branch_Count",
		"Tag_Count",
		"Has_Wiki",
		"Full_URL",
		"Created",
		"Last_Push",
		"Last_Update",
	}
}

// convertToCSVRow converts a RepositoryStats to CSV row
func convertToCSVRow(stat *models.RepositoryStats) []string {
	return []string{
		stat.Namespace,                       // Namespace
		stat.RepoName,                        // Project
		boolToString(stat.IsEmpty),           // Is_Empty
		boolToString(stat.IsFork),            // isFork
		boolToString(stat.IsArchive),         // isArchive
		fmt.Sprintf("%.0f", stat.RepoSizeMB), // Project_Size(mb) - no decimals
		fmt.Sprintf("%.0f", stat.LFSSizeMB),  // LFS_Size(mb) - no decimals
		fmt.Sprintf("%d", stat.CollaboratorCount),
		fmt.Sprintf("%d", stat.ProtectedBranchCount),
		fmt.Sprintf("%d", stat.MRReviewCount), // MR_Review_Count
		fmt.Sprintf("%d", stat.MilestoneCount),
		fmt.Sprintf("%d", stat.IssueCount),
		fmt.Sprintf("%d", stat.MRCount),              // MR_Count
		fmt.Sprintf("%d", stat.MRReviewCommentCount), // MR_Review_Comment_Count
		fmt.Sprintf("%d", stat.CommitCount),          // Commit_Count
		fmt.Sprintf("%d", stat.IssueCommentCount),    // Issue_Comment_Count
		fmt.Sprintf("%d", stat.ReleaseCount),
		fmt.Sprintf("%d", stat.BranchCount),
		fmt.Sprintf("%d", stat.TagCount),
		boolToString(stat.HasWiki),    // Has_Wiki
		stat.FullURL,                  // Full_URL
		timeToString(stat.Created),    // Created
		timeToString(stat.LastPush),   // Last_Push
		timeToString(stat.LastUpdate), // Last_Update
	}
}

// Helper functions

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func timeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ProgressReporter interface for progress reporting
type ProgressReporter interface {
	Start(total int)
	Update(current int)
	Finish()
}

// QuietProgress implements a quiet progress reporter (no output)
type QuietProgress struct{}

// NewQuietProgress creates a new quiet progress reporter
func NewQuietProgress() *QuietProgress {
	return &QuietProgress{}
}

// Start is a no-op for quiet progress
func (p *QuietProgress) Start(total int) {}

// Update is a no-op for quiet progress
func (p *QuietProgress) Update(current int) {}

// Finish is a no-op for quiet progress
func (p *QuietProgress) Finish() {}
