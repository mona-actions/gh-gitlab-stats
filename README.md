# gh-gitlab-stats

## Description

GitLab-Stats is a command-line interface that gathers GitLab metrics from a specified instance. It requires a GitLab hostname and token to authenticate to the instance. The tool outputs the results to a CSV file with a default filename of `gitlab-stats-YYYY-MM-DD-HH-MM-SS.csv`. Give it a try and see what metrics you can gather!

## Requirements

- Go 1.16 or higher

## Inputs

| Name | Description | Required | Default |
|------|-------------|----------|---------|
| `gitlab-hostname` | The hostname of the GitLab instance to gather metrics from. E.g https://gitlab.company.com | Yes | N/A |
| `token` | The token to use to authenticate to the GitLab instance. | Yes | N/A |
| `output-file` | The output file name to write the results to. | No | `gitlab-stats-YYYY-MM-DD-HH-MM-SS.csv` |

## How to Run

1. Install dependencies: `go mod download`
2. Build the tool: `go build .`
3. Run the tool: `./gh-gitlab-stats --gitlab-hostname <hostname> --token <token> --output-file <filename>`

## Usage

```
./gh-gitlab-stats --help
gh cli extension for analyzing GitLab Instance to get migration statistics of
              repositories, issues...

Usage:
  gh-gitlab-stats [flags]

Flags:
  -s, --gitlab-hostname string   The hostname of the GitLab instance to gather metrics from E.g https://gitlab.company.com
  -h, --help                     help for gh-gitlab-stats
  -f, --output-file string       The output file name to write the results to (default "gitlab-stats-YYYY-MM-DD-HH-MM-SS.csv")
  -t, --token string             The token to use to authenticate to the GitLab instance
```

## Permissions

The following permissions are required to run `gitlab-stats`, take into account that you will need to be an admin to get the full list of projects:

- `api` - Grants read-only access to the API, including all groups and projects, issues, merge requests, and the GraphQL API
- `read_user` - Grants read-only access to the authenticated user's profile through the /user API endpoint, including username, public email, and full name
- `read_repository` - Grants read-only access to repositories, including private repositories

## Output

`gitlab-stats` outputs the following metrics to a CSV file:

```csv
Namespace Name,Project_Name,Is_Empty,Last_Push,Last_Update,isFork,Repository_Size(mb),Record_Count,Collaborator_Count,Protected_Branch_Count,MR_Review_Count,Milestone_Count,Issue_Count,MergeRequest_Count,MR_Review_Comment_Count,Commit_Comment_Count,Issue_Comment_Count,Issue_Event_Count,Release_Count,Project_Count,Branch_Count,Tag_Count,Has Wiki,Full_URL,Migration_Issue
TheLeafVillage,naruto,false,N\A,2023-08-21T22:22:55Z,false,0,12,2,Protected Branch Count To be implemented," Mr Review Count To be implemented",1,3,1,1,2,2,N\A,0,N\A,2,0,true,http://gitlab.amenocal.io/theleafvillage/naruto
```

### Columns

- `Namespace_Name`: Namespace name of the Project
- `Repo_Name`: Repository name
- `Is_Empty`: Whether the repository is empty
- `Last_Push`: **To be implemented**
- `Last_Update`: Date/time when an update was last made
- `isFork`: Whether the repository is a fork
- `Repo_Size(mb)`: Size of the repository in megabytes
- `Record_Count`: Number of database records this repository represents
- `Collaborator_Count`: Number of users who are members to this repository
- `Protected_Branch_Count`: **To be implemented**
- `MR_Review_Count`: **To be implemented**
- `Milestone_Count`: Number of milestones
- `Issue_Count`: Number of issues
- `MergeRequest_Count`: Number of Merge requests
- `MR_Review_Comment_Count`: **To be implemented**
- `Commit_Comment_Count`: Number of commit comments
- `Issue_Comment_Count`: Number of issue comments
- `Issue_Event_Count`: Number of issues
- `Release_Count`: Number of releases
- `Project_Count`: "N\A"
- `Branch_Count`: Number of branches
- `Tag_Count`: Number of tags
- `Discussion_Count`: "N\A"
- `Has_Wiki`: Whether the repository has wiki feature enabled; unable to tell whether user via API
- `Full_URL`: Repository URL
- `Migration_Issue`: Indicates whether the repository might have a problem during migration due to
  - 60,000 or more number of objects being imported
  - 1.5 GB or larger size on disk

## Caveats

- Watch out for RateLimiting when running this tools. Extensive testing hasn't been done.
- Some metrics are not exported due to them not being available or need to be implemented. See in [columns](#columns) section.
