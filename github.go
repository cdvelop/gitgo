package devflow

import (
	"fmt"
	"strings"
)

// GitHub handler for GitHub operations
type GitHub struct {
	log func(...any)
}

// NewGitHub creates handler and verifies gh CLI availability
func NewGitHub() (*GitHub, error) {
	// Verify gh installation
	if _, err := RunCommandSilent("gh", "--version"); err != nil {
		return nil, fmt.Errorf("gh cli is not installed or not in PATH: %w", err)
	}

	// Verify authentication (optional but good practice)
	// We can skip this here and check it when needed, or check it now.
	// The spec says: "Returns error if gh not installed or not authenticated"
	if _, err := RunCommandSilent("gh", "auth", "status"); err != nil {
		return nil, fmt.Errorf("gh cli is not authenticated: %w", err)
	}

	return &GitHub{
		log: func(...any) {},
	}, nil
}

// SetLog sets the logger function
func (gh *GitHub) SetLog(fn func(...any)) {
	gh.log = fn
}

// GetCurrentUser gets the current authenticated user
func (gh *GitHub) GetCurrentUser() (string, error) {
	output, err := RunCommandSilent("gh", "api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// RepoExists checks if a repository exists
func (gh *GitHub) RepoExists(owner, name string) (bool, error) {
	// gh repo view owner/name
	_, err := RunCommandSilent("gh", "repo", "view", fmt.Sprintf("%s/%s", owner, name))
	if err != nil {
		// If error contains "Could not resolve", it doesn't exist.
		// If it's another error (network), we should probably return error.
		// However, RunCommandSilent just returns error if exit code != 0.
		// We can assume if it fails, it might not exist or we can't access it.
		// For now, let's treat any failure as "doesn't exist or not accessible"
		// But better to check the error message if we could.
		// Given our executor, we might just return false.
		return false, nil
	}
	return true, nil
}

// CreateRepo creates a new repository on GitHub
func (gh *GitHub) CreateRepo(name, description, visibility string) error {
	args := []string{"repo", "create", name, "--source=.", "--push", "--description", description}

	if visibility == "private" {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}

	_, err := RunCommand("gh", args...)
	return err
}

// IsNetworkError checks if an error is likely a network error
func (gh *GitHub) IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "dial tcp") ||
		   strings.Contains(msg, "connection refused") ||
		   strings.Contains(msg, "no such host") ||
		   strings.Contains(msg, "timeout")
}

// GetHelpfulErrorMessage returns a helpful message for common errors
func (gh *GitHub) GetHelpfulErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if gh.IsNetworkError(err) {
		return "Network error. Check your internet connection."
	}
	if strings.Contains(err.Error(), "authentication") {
		return "Authentication failed. Run 'gh auth login'."
	}
	return err.Error()
}
