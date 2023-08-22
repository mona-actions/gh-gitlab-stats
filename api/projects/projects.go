package projects

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetProjects(client *gitlab.Client) []*gitlab.Project {

	var projects []*gitlab.Project
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Statistics: gitlab.Bool(true),
	}

	for {
		p, response, err := client.Projects.ListProjects(opt)

		if err != nil {
			log.Fatalf("Failed to list projects: %v", err)
		}
		projects = append(projects, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return projects
}

func GetProjectMilestones(project *gitlab.Project, client *gitlab.Client) []*gitlab.Milestone {
	var milestones []*gitlab.Milestone
	opt := &gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Milestones.ListMilestones(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list milestones: %v %v", response, err)
		}
		milestones = append(milestones, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, milestone := range milestones {
		log.Println("Found milestone: ", milestone.Title)
	}

	log.Println("Number of milestones: ", len(milestones))

	return milestones

}

func GetProjectBranches(project *gitlab.Project, client *gitlab.Client) []*gitlab.Branch {
	var branches []*gitlab.Branch
	opt := &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Branches.ListBranches(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list branches: %v %v", response, err)
		}
		branches = append(branches, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, branch := range branches {
		log.Println("Found branch: ", branch.Name)
	}

	return branches
}

func GetProjectReleases(project *gitlab.Project, client *gitlab.Client) []*gitlab.Release {
	var releases []*gitlab.Release
	opt := &gitlab.ListReleasesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Releases.ListReleases(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list releases: %v %v", response, err)
		}
		releases = append(releases, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return releases
}
