package commits

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetCommitActivity(project *gitlab.Project, client *gitlab.Client) []*gitlab.Commit {
	var commits []*gitlab.Commit
	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}
	for {
		c, response, err := client.Commits.ListCommits(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list commits: %v %v", response, err)
		}
		commits = append(commits, c...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	//TODO: Need to decide if we would like to build a new struct for more readable Commit Summary or just use the gitlab.Commit struct

	return commits
}

func GetCommitComments(project *gitlab.Project, commit *gitlab.Commit, client *gitlab.Client) []*gitlab.CommitComment {
	var commitComments []*gitlab.CommitComment
	opt := &gitlab.GetCommitCommentsOptions{
		PerPage: 100,
		Page:    1,
	}

	for {
		c, response, err := client.Commits.GetCommitComments(project.ID, commit.ShortID, opt)
		if err != nil {
			log.Fatalf("Failed to list commits: %v %v", response, err)
		}

		commitComments = append(commitComments, c...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}
	//fmt.Println(commitComments)

	return commitComments
}
