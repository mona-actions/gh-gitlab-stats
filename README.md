# gh-gitlab-stats

## Description

GitLab-Stats is a command-line interface that gathers GitLab metrics from a specified instance. It requires a GitLab hostname and token to authenticate to the instance. The tool outputs the results to a CSV file with a default filename of `gitlab-stats-YYYY-MM-DD-HH-MM-SS.csv`. Give it a try and see what metrics you can gather!

## Requirements

- Go 1.16 or higher
- GitLab Server (tested on v16.2.4) (**Note:** This tool is not compatible with `GitLab.com`)

## Inputs

| Name | Description | Required | Default |
|------|-------------|----------|---------|
| `hostname` | The hostname of the GitLab instance to gather metrics from. E.g `https://gitlab.company.com` | Yes | N/A |
| `token` | The token to use to authenticate to the GitLab instance. | Yes | N/A |
| `output-file` | The output file name to write the results to. | No | `gitlab-stats-YYYY-MM-DD-HH-MM-SS.csv` |

## How to Run

1. `gh extension install mona-actions/gh-gitlab-stats`
2. Run the tool: `gh gitlab-stats --hostname <hostname> --token <token> --output-file <filename>`

## Usage

```sh
gh gitlab-stats --help
gh cli extension for analyzing GitLab Instance to get migration statistics of
              repositories, issues...

Usage:
  gh gitlab-stats [flags]

Flags:
  -s, --hostname string   The hostname of the GitLab instance to gather metrics from E.g https://gitlab.company.com
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
Namespace Name,Project_Name,Is_Empty,Last_Push,Last_Update,isFork,Repository_Size(mb),Record_Count,Collaborator_Count,Protected_Branch_Count,MR_Review_Count,Milestone_Count,Issue_Count,MergeRequest_Count,MR_Review_Comment_Count,Commit_Comment_Count,Issue_Comment_Count,Issue_Event_Count,Release_Count,Issue_Board_Count,Branch_Count,Tag_Count,Discussion_Count,Has Wiki,Full_URL,Migration_Issue
theleafvillage/hyuga,neji,false,2023-08-31T22:08:34Z,2023-08-31T22:08:33Z,false,0,2,2,1," Mr Review Count To be implemented",0,0,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/theleafvillage/hyuga/neji
thesandvillage,gara,false,2023-08-31T22:07:10Z,2023-08-31T22:07:10Z,false,0,2,1,1," Mr Review Count To be implemented",0,0,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/thesandvillage/gara
theleafvillage,repo10,false,2023-01-24T16:22:59-06:00,2023-08-24T17:57:27Z,false,0,5,2,1," Mr Review Count To be implemented",0,3,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/theleafvillage/repo10
theleafvillage,repo9,false,2023-01-24T16:22:59-06:00,2023-08-24T17:57:26Z,false,0,5,2,1," Mr Review Count To be implemented",0,3,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/theleafvillage/repo9
theleafvillage,repo8,false,2023-01-24T16:22:59-06:00,2023-08-24T17:57:26Z,false,0,5,2,1," Mr Review Count To be implemented",0,3,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/theleafvillage/repo8
theleafvillage,repo7,false,2023-01-24T16:22:59-06:00,2023-08-24T17:57:25Z,false,0,5,2,1," Mr Review Count To be implemented",0,3,0,0,0,0,N\A,0,0,1,0,N\A,false,http://gitlab-amenocal.expert-services.io/theleafvillage/repo7
```

### Columns

- `Namespace_Name`: Namespace path of the Project
- `Repo_Name`: Repository name
- `Is_Empty`: Whether the repository is empty
- `Last_Push`: Date/time when a push was last made to the default branch
- `Last_Update`: Date/time when an update was last made
- `isFork`: Whether the repository is a fork
- `Repo_Size(mb)`: Size of the repository in megabytes
- `Record_Count`: Number of database records this repository represents
- `Collaborator_Count`: Number of users who are members to this repository
- `Protected_Branch_Count`: Number of protected branches
- `MR_Review_Count`: **To be implemented**
- `Milestone_Count`: Number of milestones
- `Issue_Count`: Number of issues
- `MergeRequest_Count`: Number of Merge requests
- `MR_Review_Comment_Count`: Number of merge request comments
- `Commit_Comment_Count`: Number of commit comments
- `Issue_Comment_Count`: Number of issue comments
- `Issue_Event_Count`: "N\A"
- `Release_Count`: Number of releases
- `Issue_Board_Count`: Number of Issue Boards
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
