package devflow

// GitHubClient defines the interface for GitHub operations.
// This allows mocking the GitHub dependency in tests.
type GitHubClient interface {
	SetLog(fn func(...any))
	GetCurrentUser() (string, error)
	RepoExists(owner, name string) (bool, error)
	CreateRepo(owner, name, description, visibility string) error
	DeleteRepo(owner, name string) error
	IsNetworkError(err error) bool
	GetHelpfulErrorMessage(err error) string
}

// GitClient defines the interface for Git operations.
type GitClient interface {
	CheckRemoteAccess() error
	Push(message, tag string) (string, error)
	GetLatestTag() (string, error)
	SetLog(fn func(...any))
}
