package groups

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetGroups(client *gitlab.Client) []*gitlab.Group {
	var groups []*gitlab.Group
	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	//TODO: Check to see if pagination can be extrapolated to a function
	for {
		g, response, err := client.Groups.ListGroups(opt)

		if err != nil {
			log.Printf("Failed to list groups: %v", err)
		}
		groups = append(groups, g...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, group := range groups {
		log.Printf("Found group %s", group.Name)
	}

	return groups
}

func GetGroupsByName(client *gitlab.Client, groupName string) []*gitlab.Group {
	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Search: &groupName,
	}
	group, _, err := client.Groups.ListGroups(opt)
	if err != nil {
		log.Printf("Failed to list groups: %v", err)
		return nil
	}
	return group
}

func GetGroupsProjects(client *gitlab.Client, groups []*gitlab.Group) []*gitlab.Project {
	var projects []*gitlab.Project
	for _, group := range groups {
		opt := &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 100,
				Page:    1,
			},
		}
		for {
			p, response, err := client.Groups.ListGroupProjects(group.ID, opt)

			if err != nil {
				log.Printf("Failed to list projects: %v", err)
			}
			projects = append(projects, p...)

			if response.NextPage == 0 {
				break
			}

			opt.Page = response.NextPage
		}
	}

	return projects
}
