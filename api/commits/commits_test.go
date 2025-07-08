package commits

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetCommitActivity(t *testing.T) {
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

	// Mock the API response for listing project commits
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/repository/commits").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Commit{
			{
				ID:      "abc123",
				ShortID: "abc123",
				Title:   "Test commit 1",
			},
			{
				ID:      "def456",
				ShortID: "def456",
				Title:   "Test commit 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	commits := GetCommitActivity(project, client)

	// Verify results
	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}

	if commits[0].Title != "Test commit 1" {
		t.Errorf("Expected first commit title 'Test commit 1', got '%s'", commits[0].Title)
	}

	if commits[1].Title != "Test commit 2" {
		t.Errorf("Expected second commit title 'Test commit 2', got '%s'", commits[1].Title)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetCommitComments(t *testing.T) {
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

	// Mock project and commit
	project := &gitlab.Project{
		ID:                1,
		NameWithNamespace: "test/project",
	}
	commit := &gitlab.Commit{
		ID:      "abc123",
		ShortID: "abc123",
	}

	// Mock the API response for listing commit comments
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/repository/commits/abc123/comments").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.CommitComment{
			{
				Note: "Test commit comment 1",
			},
			{
				Note: "Test commit comment 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	comments := GetCommitComments(project, commit, client)

	// Verify results
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}

	if comments[0].Note != "Test commit comment 1" {
		t.Errorf("Expected first comment note 'Test commit comment 1', got '%s'", comments[0].Note)
	}

	if comments[1].Note != "Test commit comment 2" {
		t.Errorf("Expected second comment note 'Test commit comment 2', got '%s'", comments[1].Note)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}