package issues

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetProjectIssues(t *testing.T) {
	defer gock.Off()

	// Create a custom HTTP client that gock can intercept
	httpClient := &http.Client{}
	gock.InterceptClient(httpClient)

	// Mock GitLab client with custom HTTP client
	client, err := gitlab.NewClient("test-token",
		gitlab.WithBaseURL("https://gitlab.example.com/api/v4"),
		gitlab.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Mock project
	project := &gitlab.Project{
		ID:                1,
		NameWithNamespace: "test/project",
	}

	// Mock the API response for listing project issues
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/issues").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Issue{
			{
				ID:    1,
				IID:   1,
				Title: "Test Issue 1",
			},
			{
				ID:    2,
				IID:   2,
				Title: "Test Issue 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	issues := GetProjectIssues(project, client)

	// Verify results
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}

	if issues[0].Title != "Test Issue 1" {
		t.Errorf("Expected first issue title 'Test Issue 1', got '%s'", issues[0].Title)
	}

	if issues[1].Title != "Test Issue 2" {
		t.Errorf("Expected second issue title 'Test Issue 2', got '%s'", issues[1].Title)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetIssueComments(t *testing.T) {
	defer gock.Off()

	// Create a custom HTTP client that gock can intercept
	httpClient := &http.Client{}
	gock.InterceptClient(httpClient)

	// Mock GitLab client with custom HTTP client
	client, err := gitlab.NewClient("test-token",
		gitlab.WithBaseURL("https://gitlab.example.com/api/v4"),
		gitlab.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Mock project and issue
	project := &gitlab.Project{
		ID:                1,
		NameWithNamespace: "test/project",
	}
	issue := &gitlab.Issue{
		ID:  1,
		IID: 1,
	}

	// Mock the API response for listing issue notes
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/issues/1/notes").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Note{
			{
				ID:   1,
				Body: "Test comment 1",
			},
			{
				ID:   2,
				Body: "Test comment 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	comments := GetIssueComments(project, issue, client)

	// Verify results
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}

	if comments[0].Body != "Test comment 1" {
		t.Errorf("Expected first comment body 'Test comment 1', got '%s'", comments[0].Body)
	}

	if comments[1].Body != "Test comment 2" {
		t.Errorf("Expected second comment body 'Test comment 2', got '%s'", comments[1].Body)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetIssueBoards(t *testing.T) {
	defer gock.Off()

	// Create a custom HTTP client that gock can intercept
	httpClient := &http.Client{}
	gock.InterceptClient(httpClient)

	// Mock GitLab client with custom HTTP client
	client, err := gitlab.NewClient("test-token",
		gitlab.WithBaseURL("https://gitlab.example.com/api/v4"),
		gitlab.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Mock project
	project := &gitlab.Project{
		ID:                1,
		NameWithNamespace: "test/project",
	}

	// Mock the API response for listing issue boards
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/boards").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.IssueBoard{
			{
				ID:   1,
				Name: "Test Board 1",
			},
			{
				ID:   2,
				Name: "Test Board 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	boards := GetIssueBoards(project, client)

	// Verify results
	if len(boards) != 2 {
		t.Errorf("Expected 2 boards, got %d", len(boards))
	}

	if boards[0].Name != "Test Board 1" {
		t.Errorf("Expected first board name 'Test Board 1', got '%s'", boards[0].Name)
	}

	if boards[1].Name != "Test Board 2" {
		t.Errorf("Expected second board name 'Test Board 2', got '%s'", boards[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}
