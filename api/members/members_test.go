package members

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetProjectMembers(t *testing.T) {
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

	// Mock the API response for listing project members (all)
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/members/all").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.ProjectMember{
			{
				ID:   1,
				Name: "Test User 1",
			},
			{
				ID:   2,
				Name: "Test User 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	members := GetProjectMembers(project, client)

	// Verify results
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	if members[0].Name != "Test User 1" {
		t.Errorf("Expected first member name 'Test User 1', got '%s'", members[0].Name)
	}

	if members[1].Name != "Test User 2" {
		t.Errorf("Expected second member name 'Test User 2', got '%s'", members[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}
