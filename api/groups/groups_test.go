package groups

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetGroups(t *testing.T) {
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

	// Mock the API response for listing groups
	gock.New("https://gitlab.example.com").
		Get("/api/v4/groups").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Group{
			{
				ID:   1,
				Name: "Test Group 1",
			},
			{
				ID:   2,
				Name: "Test Group 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	groups := GetGroups(client)

	// Verify results
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	if groups[0].Name != "Test Group 1" {
		t.Errorf("Expected first group name 'Test Group 1', got '%s'", groups[0].Name)
	}

	if groups[1].Name != "Test Group 2" {
		t.Errorf("Expected second group name 'Test Group 2', got '%s'", groups[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetGroupsByName(t *testing.T) {
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

	groupName := "test-group"

	// Mock the API response for searching groups by name
	gock.New("https://gitlab.example.com").
		Get("/api/v4/groups").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		MatchParam("search", groupName).
		Reply(200).
		JSON([]*gitlab.Group{
			{
				ID:   1,
				Name: "test-group",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	groups := GetGroupsByName(client, groupName)

	// Verify results
	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if groups[0].Name != "test-group" {
		t.Errorf("Expected group name 'test-group', got '%s'", groups[0].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetGroupsProjects(t *testing.T) {
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

	// Mock groups
	groups := []*gitlab.Group{
		{
			ID:   1,
			Name: "Test Group 1",
		},
		{
			ID:   2,
			Name: "Test Group 2",
		},
	}

	// Mock the API response for listing projects in group 1
	gock.New("https://gitlab.example.com").
		Get("/api/v4/groups/1/projects").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Project{
			{
				ID:   1,
				Name: "Project 1",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Mock the API response for listing projects in group 2
	gock.New("https://gitlab.example.com").
		Get("/api/v4/groups/2/projects").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Project{
			{
				ID:   2,
				Name: "Project 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	projects := GetGroupsProjects(client, groups)

	// Verify results
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Project 1" {
		t.Errorf("Expected first project name 'Project 1', got '%s'", projects[0].Name)
	}

	if projects[1].Name != "Project 2" {
		t.Errorf("Expected second project name 'Project 2', got '%s'", projects[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}
