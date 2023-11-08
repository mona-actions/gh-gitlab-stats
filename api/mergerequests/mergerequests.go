package mergerequests

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetMergeRequests(project *gitlab.Project, client *gitlab.Client) []*gitlab.MergeRequest {
	var mergeRequests []*gitlab.MergeRequest
	opt := &gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.MergeRequests.ListProjectMergeRequests(project.ID, opt)
		if err != nil {
			log.Printf("Failed to list MergeRequests for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		mergeRequests = append(mergeRequests, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	return mergeRequests
}

func GetMergeRequestComments(project *gitlab.Project, mr *gitlab.MergeRequest, client *gitlab.Client) []*gitlab.Note {
	var mrComments []*gitlab.Note
	opt := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		p, response, err := client.Notes.ListMergeRequestNotes(project.ID, mr.IID, opt)
		if err != nil {
			log.Printf("Failed to list MergeRequests Comments for: %v, response: %v error: %v", project.NameWithNamespace, response, err)
		}
		mrComments = append(mrComments, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}
	return mrComments

}
