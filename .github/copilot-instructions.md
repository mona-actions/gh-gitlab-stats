# GitHub CLI Extension Development Best Practices

This document provides comprehensive guidelines for developing high-quality GitHub CLI extensions using Go and modern development practices.

## Technology Stack

### Core Technologies

- **Programming Language**: Go (version 1.20+)
  - **Cobra**: Command-line interface framework for building CLI applications with subcommands, flags, and help generation
  - **Viper**: Configuration management library supporting environment variables, config files, and command-line flags
  - **go-github**: Official Go client library for GitHub REST API v3 and v4
  - **githubv4**: GraphQL client for GitHub API v4 (for complex queries and better performance)
- **GitHub CLI**: Foundation for extension development, providing authentication and GitHub integration
- **GitHub API**: Primary data source for GitHub resources (REST API v3, GraphQL API v4)

### Optional UI Libraries

- **tview**: Terminal-based user interface library for building interactive TUIs
- **tcell**: Low-level terminal handling (used by tview)
- **go-github-ratelimit**: Rate limiting wrapper for GitHub API clients

## Architecture Principles

### 1. Separation of Concerns

- **Commands** (`cmd/`): Handle CLI parsing, validation, and orchestration
- **Business Logic** (`internal/`): Core application functionality and domain models
- **External Integrations** (`internal/api/`): GitHub API clients and external service interactions
- **Configuration** (`internal/config/`): Application configuration and environment management
- **User Interface** (`internal/ui/`): Output formatting, TUI components, and user interaction

### 2. Dependency Injection

- Use interfaces to define contracts between layers
- Implement dependency injection for testability and modularity
- Avoid global state and singletons where possible

### 3. Error Handling

- Use Go's idiomatic error handling with custom error types
- Wrap errors with context using `fmt.Errorf` or dedicated error libraries
- Provide meaningful error messages to end users

## Project Structure Best Practices

```
project-root/
├── main.go                    # Entry point
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── README.md                  # Project documentation
├── LICENSE                    # License file
├── .github/
│   ├── workflows/             # CI/CD pipelines
│   └── copilot-instructions.md
├── cmd/                       # CLI command definitions
│   ├── root.go               # Root command and global flags
│   └── subcommand.go         # Individual subcommands
├── internal/                  # Private application code
│   ├── api/                  # GitHub API clients
│   │   ├── client.go         # API client interface and implementation
│   │   ├── github.go         # GitHub-specific API methods
│   │   └── types.go          # API response types
│   ├── config/               # Configuration management
│   │   ├── config.go         # Configuration structure and loading
│   │   └── validation.go     # Configuration validation
│   ├── models/               # Domain models and business entities
│   │   └── types.go
│   ├── services/             # Business logic services
│   │   └── service.go
│   └── ui/                   # User interface components
│       ├── formatter.go      # Output formatting
│       ├── table.go          # Table rendering
│       └── tui.go            # Terminal UI components
├── pkg/                      # Public API (if applicable)
│   └── client/               # Public client library
└── tests/                    # Integration and end-to-end tests
    ├── fixtures/             # Test data
    └── integration/          # Integration tests
```

## Development Guidelines

### Code Organization

1. **Use the `internal/` directory** for all private application code that shouldn't be imported by other projects
2. **Implement clean architecture** with clear separation between presentation, business logic, and data layers
3. **Keep commands thin** - CLI commands should only handle parsing, validation, and orchestration
4. **Use interfaces extensively** for all external dependencies (GitHub API, file system, etc.)
5. **Group related functionality** into cohesive packages with single responsibilities

### API Client Design

```go
// Define interfaces for testability
type GitHubClient interface {
    GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
    ListIssues(ctx context.Context, owner, repo string, opts *ListOptions) ([]*Issue, error)
}

// Implement rate limiting and error handling
type githubClient struct {
    restClient    *github.Client
    graphqlClient *githubv4.Client
    rateLimiter   github_ratelimit.Limiter
}
```

### Configuration Management

```go
// Use Viper for flexible configuration
type Config struct {
    GitHub struct {
        Token        string `mapstructure:"token"`
        BaseURL      string `mapstructure:"base_url"`
        Organization string `mapstructure:"organization"`
    } `mapstructure:"github"`

    Output struct {
        Format string `mapstructure:"format"`
        Quiet  bool   `mapstructure:"quiet"`
    } `mapstructure:"output"`
}

// Support multiple configuration sources
func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("$HOME/.gh-extension")
    viper.AddConfigPath(".")

    // Environment variables
    viper.SetEnvPrefix("GH_EXT")
    viper.AutomaticEnv()

    // Read configuration
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }

    var config Config
    return &config, viper.Unmarshal(&config)
}
```

### Command Structure

```go
// Keep commands focused on CLI concerns
func NewListCommand(service ListService) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List resources",
        RunE: func(cmd *cobra.Command, args []string) error {
            opts, err := parseListOptions(cmd)
            if err != nil {
                return err
            }

            results, err := service.List(cmd.Context(), opts)
            if err != nil {
                return fmt.Errorf("failed to list resources: %w", err)
            }

            return renderResults(cmd.OutOrStdout(), results, opts.Format)
        },
    }

    // Add flags
    cmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
    cmd.Flags().StringP("filter", "", "", "Filter results")

    return cmd
}
```

### Error Handling Patterns

```go
// Define custom error types
type APIError struct {
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}

// Wrap errors with context
func (c *githubClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
    repo, resp, err := c.restClient.Repositories.Get(ctx, owner, repo)
    if err != nil {
        if resp != nil && resp.StatusCode == 404 {
            return nil, &APIError{
                StatusCode: 404,
                Message:    fmt.Sprintf("repository %s/%s not found", owner, repo),
                Err:        err,
            }
        }
        return nil, fmt.Errorf("failed to get repository %s/%s: %w", owner, repo, err)
    }

    return convertRepository(repo), nil
}
```

### Testing Strategy

1. **Unit Tests**: Test business logic and individual functions
2. **Integration Tests**: Test API interactions with real or mock GitHub API
3. **End-to-End Tests**: Test complete CLI workflows
4. **Table-Driven Tests**: Use Go's table-driven test pattern for comprehensive coverage

```go
// Example unit test with mocking
func TestListService_List(t *testing.T) {
    tests := []struct {
        name        string
        setupMock   func(*MockGitHubClient)
        options     ListOptions
        want        []Resource
        wantErr     bool
    }{
        {
            name: "successful list",
            setupMock: func(m *MockGitHubClient) {
                m.EXPECT().ListRepositories(gomock.Any(), gomock.Any()).
                    Return([]*Repository{{Name: "test"}}, nil)
            },
            options: ListOptions{Owner: "testorg"},
            want:    []Resource{{Name: "test"}},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockClient := NewMockGitHubClient(ctrl)
            tt.setupMock(mockClient)

            service := NewListService(mockClient)
            got, err := service.List(context.Background(), tt.options)

            if (err != nil) != tt.wantErr {
                t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("List() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Performance Considerations

1. **Use GraphQL** for complex queries that require multiple REST API calls
2. **Implement pagination** for large result sets
3. **Add rate limiting** to respect GitHub API limits
4. **Use context** for cancellation and timeouts
5. **Cache frequently accessed data** when appropriate

### Security Best Practices

1. **Never hardcode tokens** in source code
2. **Use GitHub CLI's authentication** when possible
3. **Validate all user inputs** to prevent injection attacks
4. **Use HTTPS** for all API communications
5. **Handle sensitive data** appropriately (don't log tokens, etc.)

### Documentation Standards

1. **Package documentation**: Every package should have a doc.go file
2. **Function documentation**: Document all exported functions and types
3. **Example usage**: Provide examples in documentation and README
4. **API documentation**: Document command flags and expected behavior
5. **Architecture decisions**: Document significant design decisions