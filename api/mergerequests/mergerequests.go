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
			log.Fatalf("Failed to list merge requests: %v %v", response, err)
		}
		mergeRequests = append(mergeRequests, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	for _, mergeRequest := range mergeRequests {
		log.Println("Found merge request: ", mergeRequest.ID, mergeRequest.Title, mergeRequest.Author.Username)
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
			log.Fatalf("Failed to list merge request comments: %v %v", response, err)
		}
		mrComments = append(mrComments, p...)

		if response.NextPage == 0 {
			break
		}

		opt.Page = response.NextPage
	}

	log.Println("No. issue comments: ", len(mrComments))
	return mrComments

}
