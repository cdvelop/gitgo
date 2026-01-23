package devflow

// MockGitHubAuth is a mock implementation of GitHubAuthenticator for testing.
type MockGitHubAuth struct {
	log             func(...any)
	EnsureAuthError error // Set this to simulate auth failure
}

// NewMockGitHubAuth creates a new mock authenticator.
func NewMockGitHubAuth() *MockGitHubAuth {
	return &MockGitHubAuth{
		log: func(...any) {},
	}
}

// Name returns the handler name for TUI display.
func (m *MockGitHubAuth) Name() string {
	return "GitHub Auth (Mock)"
}

// SetLog sets the logger function.
func (m *MockGitHubAuth) SetLog(fn func(...any)) {
	if fn != nil {
		m.log = fn
	}
}

// EnsureGitHubAuth simulates the authentication process.
func (m *MockGitHubAuth) EnsureGitHubAuth() error {
	return m.EnsureAuthError
}
