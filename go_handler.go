package devflow

import (
	"fmt"
	"strings"
)

// Go handler for Go operations
type Go struct {
	git *Git
	log func(...any)
}

// NewGo creates a new Go handler
func NewGo(gitHandler *Git) *Go {
	return &Go{
		git: gitHandler,
		log: func(...any) {}, // default no-op
	}
}

// SetLog sets the logger function
func (g *Go) SetLog(fn func(...any)) {
	g.log = fn
}

// Push executes the complete workflow for Go projects
// Parameters:
//
//	message: Commit message
//	tag: Optional tag
//	skipTests: If true, skips tests
//	skipRace: If true, skips race tests
//	searchPath: Path to search for dependent modules (default: "..")
func (g *Go) Push(message, tag string, skipTests, skipRace bool, searchPath string) (string, error) {
	// Default values
	if message == "" {
		message = "auto update Go package"
	}

	if searchPath == "" {
		searchPath = ".."
	}

	summary := []string{}

	// 1. Verify go.mod
	if err := g.verify(); err != nil {
		return "", fmt.Errorf("go mod verify failed: %w", err)
	}

	// 2. Run tests (if not skipped)
	if !skipTests {
		testSummary, err := g.Test(false) // quiet mode
		if err != nil {
			return "", fmt.Errorf("tests failed: %w", err)
		}
		summary = append(summary, testSummary)
	} else {
		summary = append(summary, "Tests skipped")
	}

	// 3. Execute git push workflow
	pushSummary, err := g.git.Push(message, tag)
	if err != nil {
		return "", fmt.Errorf("push workflow failed: %w", err)
	}
	summary = append(summary, pushSummary)

	// 4. Get created tag
	latestTag, err := g.git.GetLatestTag()
	if err != nil {
		summary = append(summary, fmt.Sprintf("Warning: could not get latest tag: %v", err))
		// Not fatal error
	}

	// 5. Get module name
	modulePath, err := g.getModulePath()
	if err != nil {
		summary = append(summary, fmt.Sprintf("Warning: could not get module path: %v", err))
		return strings.Join(summary, ", "), nil
	}

	// 6. Update dependent modules
	updated, err := g.updateDependents(modulePath, latestTag, searchPath)
	if err != nil {
		summary = append(summary, fmt.Sprintf("Warning: failed to update dependents: %v", err))
		// Not fatal error
	}
	if updated > 0 {
		summary = append(summary, fmt.Sprintf("✅ Updated modules: %d", updated))
	}

	return strings.Join(summary, ", "), nil
}
