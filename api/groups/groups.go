package groups

import (
	"fmt"
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
			log.Fatalf("Failed to list groups: %v", err)
		}
		groups = append(groups, g...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, group := range groups {
		fmt.Println("Found group", group.Name)
	}

	return groups
}
