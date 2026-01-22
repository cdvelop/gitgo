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
