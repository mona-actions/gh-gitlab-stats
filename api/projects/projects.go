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
			log.Printf("Failed to list Project Milestones for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		milestones = append(milestones, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

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
			log.Printf("Failed to list Project Branches for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		branches = append(branches, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
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
			log.Printf("Failed to list Project Releases for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		releases = append(releases, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return releases
}

func GetProjectWikis(project *gitlab.Project, client *gitlab.Client) []*gitlab.Wiki {
	var wikis []*gitlab.Wiki
	opt := &gitlab.ListWikisOptions{
		WithContent: gitlab.Bool(true),
	}
	p, response, err := client.Wikis.ListWikis(project.ID, opt)
	if err != nil {
		log.Printf("Failed to list wikis for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
	}
	wikis = append(wikis, p...)

	return wikis
}
