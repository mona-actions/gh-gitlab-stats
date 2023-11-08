package issues

import (
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
			log.Printf("Failed to list issues for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		issues = append(issues, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

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
			log.Printf("Failed to list issues comments for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		issueComments = append(issueComments, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return issueComments

}

func GetIssueBoards(project *gitlab.Project, client *gitlab.Client) []*gitlab.IssueBoard {
	var issueBoards []*gitlab.IssueBoard
	opt := &gitlab.ListIssueBoardsOptions{
		PerPage: 100,
		Page:    1,
	}

	for {
		p, response, err := client.Boards.ListIssueBoards(project.ID, opt)
		if err != nil {
			log.Printf("Failed to list issues boards for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		issueBoards = append(issueBoards, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return issueBoards

}
