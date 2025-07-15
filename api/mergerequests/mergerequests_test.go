package mergerequests

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetMergeRequests(t *testing.T) {
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

	// Mock the API response for listing project merge requests
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/merge_requests").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.MergeRequest{
			{
				ID:    1,
				IID:   1,
				Title: "Test MR 1",
			},
			{
				ID:    2,
				IID:   2,
				Title: "Test MR 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	mergeRequests := GetMergeRequests(project, client)

	// Verify results
	if len(mergeRequests) != 2 {
		t.Errorf("Expected 2 merge requests, got %d", len(mergeRequests))
	}

	if mergeRequests[0].Title != "Test MR 1" {
		t.Errorf("Expected first merge request title 'Test MR 1', got '%s'", mergeRequests[0].Title)
	}

	if mergeRequests[1].Title != "Test MR 2" {
		t.Errorf("Expected second merge request title 'Test MR 2', got '%s'", mergeRequests[1].Title)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetMergeRequestComments(t *testing.T) {
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

	// Mock project and merge request
	project := &gitlab.Project{
		ID:                1,
		NameWithNamespace: "test/project",
	}
	mr := &gitlab.MergeRequest{
		ID:  1,
		IID: 1,
	}

	// Mock the API response for listing merge request notes
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/merge_requests/1/notes").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Note{
			{
				ID:   1,
				Body: "Test MR comment 1",
			},
			{
				ID:   2,
				Body: "Test MR comment 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	comments := GetMergeRequestComments(project, mr, client)

	// Verify results
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}

	if comments[0].Body != "Test MR comment 1" {
		t.Errorf("Expected first comment body 'Test MR comment 1', got '%s'", comments[0].Body)
	}

	if comments[1].Body != "Test MR comment 2" {
		t.Errorf("Expected second comment body 'Test MR comment 2', got '%s'", comments[1].Body)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}
