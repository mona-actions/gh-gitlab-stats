package projects

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/xanzy/go-gitlab"
)

func TestGetProjects(t *testing.T) {
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

	// Mock the API response for listing projects
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		MatchParam("statistics", "true").
		Reply(200).
		JSON([]*gitlab.Project{
			{
				ID:   1,
				Name: "Test Project 1",
			},
			{
				ID:   2,
				Name: "Test Project 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	projects := GetProjects(client)

	// Verify results
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].Name != "Test Project 1" {
		t.Errorf("Expected first project name 'Test Project 1', got '%s'", projects[0].Name)
	}

	if projects[1].Name != "Test Project 2" {
		t.Errorf("Expected second project name 'Test Project 2', got '%s'", projects[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetProject(t *testing.T) {
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
		ID:   1,
		Name: "Test Project",
	}

	// Mock the API response for getting a specific project
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1").
		MatchParam("statistics", "true").
		Reply(200).
		JSON(&gitlab.Project{
			ID:   1,
			Name: "Test Project Updated",
		})

	// Call the function
	updatedProject := GetProject(project, client)

	// Verify results
	if updatedProject.Name != "Test Project Updated" {
		t.Errorf("Expected project name 'Test Project Updated', got '%s'", updatedProject.Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetProjectMilestones(t *testing.T) {
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

	// Mock the API response for listing project milestones
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/milestones").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Milestone{
			{
				ID:    1,
				Title: "Milestone 1",
			},
			{
				ID:    2,
				Title: "Milestone 2",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	milestones := GetProjectMilestones(project, client)

	// Verify results
	if len(milestones) != 2 {
		t.Errorf("Expected 2 milestones, got %d", len(milestones))
	}

	if milestones[0].Title != "Milestone 1" {
		t.Errorf("Expected first milestone title 'Milestone 1', got '%s'", milestones[0].Title)
	}

	if milestones[1].Title != "Milestone 2" {
		t.Errorf("Expected second milestone title 'Milestone 2', got '%s'", milestones[1].Title)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetProjectBranches(t *testing.T) {
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

	// Mock the API response for listing project branches
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/repository/branches").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Branch{
			{
				Name: "main",
			},
			{
				Name: "develop",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	branches := GetProjectBranches(project, client)

	// Verify results
	if len(branches) != 2 {
		t.Errorf("Expected 2 branches, got %d", len(branches))
	}

	if branches[0].Name != "main" {
		t.Errorf("Expected first branch name 'main', got '%s'", branches[0].Name)
	}

	if branches[1].Name != "develop" {
		t.Errorf("Expected second branch name 'develop', got '%s'", branches[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetProjectReleases(t *testing.T) {
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

	// Mock the API response for listing project releases
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/releases").
		MatchParam("page", "1").
		MatchParam("per_page", "100").
		Reply(200).
		JSON([]*gitlab.Release{
			{
				Name:    "v1.0.0",
				TagName: "v1.0.0",
			},
			{
				Name:    "v2.0.0",
				TagName: "v2.0.0",
			},
		}).
		AddHeader("X-Next-Page", "").
		AddHeader("X-Total-Pages", "1")

	// Call the function
	releases := GetProjectReleases(project, client)

	// Verify results
	if len(releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(releases))
	}

	if releases[0].Name != "v1.0.0" {
		t.Errorf("Expected first release name 'v1.0.0', got '%s'", releases[0].Name)
	}

	if releases[1].Name != "v2.0.0" {
		t.Errorf("Expected second release name 'v2.0.0', got '%s'", releases[1].Name)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}

func TestGetProjectWikis(t *testing.T) {
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

	// Mock the API response for listing project wikis
	gock.New("https://gitlab.example.com").
		Get("/api/v4/projects/1/wikis").
		MatchParam("with_content", "true").
		Reply(200).
		JSON([]*gitlab.Wiki{
			{
				Slug:  "home",
				Title: "Home",
			},
			{
				Slug:  "about",
				Title: "About",
			},
		})

	// Call the function
	wikis := GetProjectWikis(project, client)

	// Verify results
	if len(wikis) != 2 {
		t.Errorf("Expected 2 wikis, got %d", len(wikis))
	}

	if wikis[0].Title != "Home" {
		t.Errorf("Expected first wiki title 'Home', got '%s'", wikis[0].Title)
	}

	if wikis[1].Title != "About" {
		t.Errorf("Expected second wiki title 'About', got '%s'", wikis[1].Title)
	}

	// Verify all mocks were called
	if !gock.IsDone() {
		t.Error("Not all HTTP mocks were called")
	}
}
