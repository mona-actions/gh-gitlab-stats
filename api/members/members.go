package members

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetProjectMembers(project *gitlab.Project, client *gitlab.Client) []*gitlab.ProjectMember {
	var projectMembers []*gitlab.ProjectMember
	opt := &gitlab.ListProjectMembersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.ProjectMembers.ListAllProjectMembers(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list project members: %v %v", response, err)
		}
		projectMembers = append(projectMembers, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, projectMember := range projectMembers {
		log.Println("Found project member: ", projectMember.Name, projectMember.Username)
	}

	return projectMembers
}
