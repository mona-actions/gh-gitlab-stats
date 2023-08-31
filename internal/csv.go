package internal

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"
)

func CreateCSV(data [][]string, filename string) {
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

func ConvertToCSVFormat(projects []*ProjectSummary) [][]string {
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
			strconv.Itoa(project.ProtectedBranchCount),
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
