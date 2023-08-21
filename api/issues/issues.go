package issues

import (
	"fmt"
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetProjectIssues(project *gitlab.Project, client *gitlab.Client) []*gitlab.Issue {
	var issues []*gitlab.Issue
	opt := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Issues.ListProjectIssues(project.ID, opt)
		if err != nil {
			log.Fatalf("Failed to list issues: %v %v", response, err)
		}
		issues = append(issues, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, issue := range issues {
		log.Println("Found issue: ", issue.Title)
	}

	fmt.Println("No. issues: ", len(issues))

	return issues
}

func GetIssueComments(project *gitlab.Project, issue *gitlab.Issue, client *gitlab.Client) []*gitlab.Note {
	var issueComments []*gitlab.Note
	opt := &gitlab.ListIssueNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Notes.ListIssueNotes(project.ID, issue.IID, opt)
		if err != nil {
			log.Fatalf("Failed to list issue comments: %v %v", response, err)
		}
		issueComments = append(issueComments, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	log.Println("No. issue comments: ", len(issueComments))
	return issueComments

}
