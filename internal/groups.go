package internal

import (
	"log"
	"strings"

	"github.com/mona-actions/gh-gitlab-stats/api/groups"
	"github.com/xanzy/go-gitlab"
)

func GetGroupsFromNames(client *gitlab.Client, groupNames string) []*gitlab.Group {
	var gitlabGroups []*gitlab.Group
	groupNames = strings.ReplaceAll(groupNames, " ", "")

	groupNamesSlice := strings.Split(groupNames, ",")
	for _, groupName := range groupNamesSlice {
		group := groups.GetGroupsByName(client, groupName)
		if group == nil {
			log.Printf("Group %s not found", groupName)
		}
		gitlabGroups = append(gitlabGroups, group...)
	}
	return gitlabGroups
}
