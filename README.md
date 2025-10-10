# GitLab Repository Statistics

A GitHub CLI extension for scanning GitLab instances and generating comprehensive repository statistics reports. This tool provides GitLab equivalent functionality to GitHub's repository inventory tools, generating CSV output compatible with GitHub analysis workflows.

## Features

- 🔍 **Comprehensive Scanning**: Scan GitLab.com or self-hosted GitLab instances
- 📊 **Detailed Statistics**: Collect repository metadata, collaboration metrics, and activity data
- 💬 **Comment Tracking**: Counts comments on merge requests, issues, and commits
- 👍 **Review Metrics**: Tracks merge request reviews and approvals
- 📦 **LFS Support**: Reports Git LFS storage usage per project
- 🌳 **Wiki Detection**: Verifies actual wiki content (not just enabled status)
- 🎯 **Direct REST API**: Uses GitLab REST API directly for full transparency and control
- 📈 **Real-time Progress**: Enhanced logging shows detailed progress for each project
- 🔒 **Secure**: Uses GitLab personal access tokens for authentication
- 📦 **Zero External Dependencies**: Built using only Go standard library for API calls

## Installation

### Prerequisites

- Go 1.21 or later
- GitLab personal access token with appropriate permissions

### Install from Source

```bash
git clone https://github.com/mona-actions/gh-gitlab-stats.git
cd gh-gitlab-stats
go build -o gh-gitlab-stats .
```

### GitHub CLI Extension

If you're using this as a GitHub CLI extension:

```bash
# Install as a GitHub CLI extension (if publishing to GitHub)
gh extension install mona-actions/gh-gitlab-stats
```

## Quick Start

### 1. Create a GitLab Personal Access Token

1. Go to GitLab → Settings → Access Tokens
2. Create a token with the following scopes:
   - `read_api`
   - `read_repository`

### 2. Run Your First Scan

```bash
# Scan GitLab.com (all accessible projects)
./gh-gitlab-stats --hostname gitlab.com --token YOUR_GITLAB_TOKEN

# Scan self-hosted GitLab
./gh-gitlab-stats --hostname gitlab.company.com --token YOUR_GITLAB_TOKEN

# Specify output file
./gh-gitlab-stats --hostname gitlab.com --token YOUR_GITLAB_TOKEN --output my-stats.csv
```

## Usage

### Basic Commands

```bash
# Show help
./gh-gitlab-stats --help

# Scan with verbose output
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --verbose

# Use different output format
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --format json --output report.json

# Specify custom output file
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --output my-report.csv
```

### Command-Line Options

| Flag              | Description                                   | Default      |
| ----------------- | --------------------------------------------- | ------------ |
| `--token, -t`     | GitLab access token (required)                |              |
| `--hostname, -H`  | GitLab hostname                               | `gitlab.com` |
| `--output, -O`    | Output format (CSV or Table)                  | `CSV`        |
| `--debug, -d`     | Enable debug logging with detailed progress   | `false`      |
| `--namespace, -n` | GitLab namespace/group to analyze             |              |
| `--input, -i`     | File with list of namespaces (one per line)   |              |
| `--repo-list`     | File with list of repositories (one per line) |              |

## Configuration

### Environment Variables

You can set the GitLab token via environment variable:

```bash
export GITLAB_TOKEN="your-token-here"
./gh-gitlab-stats --hostname gitlab.com
```

## Output Format

The tool generates CSV output with comprehensive GitLab project statistics:

| Column                    | Type      | Description                                  | Data Source                          |
| ------------------------- | --------- | -------------------------------------------- | ------------------------------------ |
| `Namespace`               | String    | Full namespace path (e.g., "group/subgroup") | API: `path_with_namespace`           |
| `Project`                 | String    | Project name                                 | API: `name`                          |
| `Is_Empty`                | Boolean   | Whether repository is empty                  | API: `empty_repo`                    |
| `isFork`                  | Boolean   | Whether project is a fork                    | API: `forked_from_project`           |
| `isArchive`               | Boolean   | Whether project is archived                  | API: `archived`                      |
| `Project_Size(mb)`        | Number    | Repository size in megabytes                 | API: `statistics.repository_size`    |
| `LFS_Size(mb)`            | Number    | Git LFS storage size in megabytes            | API: `statistics.lfs_objects_size`   |
| `Collaborator_Count`      | Integer   | Number of project members                    | API: `/members/all` endpoint         |
| `Protected_Branch_Count`  | Integer   | Number of protected branches (estimated)     | Computed from branch count           |
| `MR_Review_Count`         | Integer   | Number of merge request reviews/approvals    | API: MR `upvotes` + `approved_by`    |
| `Milestone_Count`         | Integer   | Number of milestones                         | API: `/milestones` endpoint          |
| `Issue_Count`             | Integer   | Number of issues (open)                      | API: `open_issues_count`             |
| `MR_Count`                | Integer   | Number of merge requests (all states)        | API: `/merge_requests` endpoint      |
| `MR_Review_Comment_Count` | Integer   | Total comments on all merge requests         | API: MR `user_notes_count` sum       |
| `Commit_Count`            | Integer   | Total number of commits                      | API: `statistics.commit_count`       |
| `Commit_Comment_Count`    | Integer   | Number of commit comments                    | Computed (typically 0)               |
| `Issue_Comment_Count`     | Integer   | Total comments on all issues                 | API: Issue `user_notes_count` sum    |
| `Release_Count`           | Integer   | Number of releases                           | API: `/releases` endpoint            |
| `Branch_Count`            | Integer   | Number of branches                           | API: `/repository/branches` endpoint |
| `Tag_Count`               | Integer   | Number of tags                               | API: `/repository/tags` endpoint     |
| `Has_Wiki`                | Boolean   | Whether wiki has actual content              | API: `/wikis` endpoint (verified)    |
| `Full_URL`                | String    | Full web URL to the project                  | API: `web_url`                       |
| `Created`                 | Timestamp | Project creation date/time (RFC3339)         | API: `created_at`                    |
| `Last_Push`               | Timestamp | Last push/activity date/time (RFC3339)       | API: `last_activity_at`              |
| `Last_Update`             | Timestamp | Last update date/time (RFC3339)              | API: `last_activity_at`              |

### Data Types

- **String**: Text values (UTF-8 encoded)
- **Boolean**: `true` or `false`
- **Integer**: Whole numbers (0, 1, 2, ...)
- **Number**: Decimal numbers (0.0, 25.5, 1024.8)
- **Timestamp**: ISO 8601 / RFC3339 format (e.g., `2023-10-10T15:30:00Z`)

### Sample Output

```csv
Namespace,Project,Is_Empty,isFork,isArchive,Project_Size(mb),LFS_Size(mb),Collaborator_Count,Protected_Branch_Count,MR_Review_Count,Milestone_Count,Issue_Count,MR_Count,MR_Review_Comment_Count,Commit_Count,Commit_Comment_Count,Issue_Comment_Count,Release_Count,Branch_Count,Tag_Count,Has_Wiki,Full_URL,Created,Last_Push,Last_Update
mygroup,awesome-project,false,false,false,250,1024,8,2,12,3,23,15,45,150,0,128,2,15,8,true,https://gitlab.com/mygroup/awesome-project,2023-01-15T10:00:00Z,2023-10-10T15:30:00Z,2023-10-10T15:30:00Z
mygroup/subgroup,another-project,false,true,false,150,0,5,1,5,1,8,5,22,85,0,35,1,8,3,false,https://gitlab.com/mygroup/subgroup/another-project,2023-03-20T14:22:00Z,2023-10-09T08:15:00Z,2023-10-09T08:15:00Z
```

## Examples

### Scan GitLab Instances

```bash
# Scan all accessible projects on GitLab.com
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN

# Scan with detailed debug logging
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --debug

# Scan self-hosted GitLab instance
./gh-gitlab-stats \
  --hostname gitlab.company.com \
  --token $GITLAB_TOKEN

# Scan specific namespace/group
./gh-gitlab-stats \
  --hostname gitlab.com \
  --token $GITLAB_TOKEN \
  --namespace mygroup/subgroup
```

### Output Formats

```bash
# CSV output (default) - saved to timestamped file
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --output CSV

# Table output (console display)
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --output Table
```

### Progress Monitoring

**Normal Mode (Compact Progress)**
```bash
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN
```
```
🔍 Discovering projects...
✓ Found 25 projects to scan

[5/25] Scanning projects... Current: group/subgroup | my-repository

═══════════════════════════════════════════════════════════════
                    SCAN COMPLETE
═══════════════════════════════════════════════════════════════
  Total projects found:     25
  Successfully processed:   25
  Errors encountered:       0
  Duration:                 2m15s
  Average time per project: 5.4s
═══════════════════════════════════════════════════════════════
```

**Debug Mode (Detailed Progress)**
```bash
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --debug
```
```
🔍 Discovering projects...
✓ Found 25 projects to scan
  Using 5 parallel workers for scanning

  → Processing: group/subgroup/project (ID: 12345)
    Fetching detailed statistics...
    ✓ Retrieved: branches(15), tags(8), members(5), issues(23), MRs(12)
    ✓ Reviews: MR Reviews(12) | Commits(150)
    ✓ Comments: MR(45), Issue(128), Commit(0)

[5/25] ✓ Scanned: group/subgroup/project
    Size: 250 MB | LFS: 1024 MB | Commits: 150 | Issues: 23 | MRs: 12 | Branches: 15 | Tags: 8
```

### API Efficiency

The tool makes efficient API calls to minimize rate limiting:
- **Pagination**: Fetches data in pages of 100 items
- **Header Counts**: Uses `X-Total` headers when available
- **Parallel Processing**: Scans up to 5 projects simultaneously
- **Sampling**: For large projects (>1000 MRs/issues), limits to first 1000

## Troubleshooting

### Common Issues

**Authentication Error**
```
Error: GitLab token is required
```
- Ensure you provide a valid GitLab personal access token
- Check token has required scopes: `read_api`, `read_repository`

**Connection Issues**
```
Error: failed to connect to GitLab
```
- Verify the hostname is correct (without `https://` prefix)
- Check network connectivity to the GitLab instance
- Ensure the GitLab instance is accessible

### Debug Mode

```bash
# Enable debug output to see detailed progress
./gh-gitlab-stats --hostname gitlab.com --token $GITLAB_TOKEN --debug
```

## Architecture

The tool follows clean architecture principles with direct REST API integration:

```
├── cmd/                    # CLI commands (Cobra)
│   └── root.go            # Root command with scan logic
├── internal/
│   ├── api/               # GitLab REST API client
│   │   ├── rest_client.go # Direct HTTP/REST implementation
│   │   └── types.go       # API response types
│   ├── models/            # Domain models
│   │   └── types.go       # RepositoryStats, ScanOptions
│   ├── services/          # Business logic
│   │   └── scanner.go     # Project scanning service
│   └── ui/                # Output formatting
│       └── formatter.go   # CSV/JSON/YAML formatters
└── main.go                # Entry point
```

### Key Components

- **REST Client**: Direct HTTP calls to GitLab REST API v4 using Go standard library
  - Fetches project metadata, statistics, and counts
  - Implements efficient pagination and header-based counting
  - Verifies wiki content, counts comments, and tracks reviews
- **Scanner Service**: Orchestrates project discovery and statistics collection
  - Parallel processing with worker pools (5 concurrent workers)
  - Real-time progress reporting
  - Error handling and recovery
- **Formatters**: Convert statistics to CSV or Table output
- **Progress Reporters**: Console and quiet modes for different use cases
- **Zero Dependencies**: Uses only Go standard library for API calls (no external GitLab SDK)

### Statistics Collection Flow

1. **Discovery**: Fetch all accessible projects via `/projects` endpoint
2. **Parallel Scanning**: Process projects using worker pool
3. **Per Project**:
   - Fetch detailed statistics with `statistics=true`
   - Count branches, tags, members, milestones, releases
   - Sum comments from MRs and issues
   - Count MR reviews/approvals
   - Verify wiki content
4. **Output**: Format and write results to CSV/Table

### Development Setup

```bash
git clone https://github.com/mona-actions/gh-gitlab-stats.git
cd gh-gitlab-stats
go mod tidy
go build -o gh-gitlab-stats .
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) file for details.
