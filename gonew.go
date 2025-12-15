package devflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoNew orchestrator
type GoNew struct {
	git    *Git
	github *GitHub
	goH    *Go
	log    func(...any)
}

// NewProjectOptions options for creating a new project
type NewProjectOptions struct {
	Name        string // Required, must be valid (alphanumeric, dash, underscore only)
	Description string // Required, max 350 chars
	Visibility  string // "public" or "private" (default: "public")
	Directory   string // Supports ~/path, ./path, /abs/path (default: ./{Name})
	LocalOnly   bool   // If true, skip remote creation
	License     string // Default "MIT"
}

// NewGoNew creates orchestrator (all handlers must be initialized)
func NewGoNew(git *Git, github *GitHub, goHandler *Go) *GoNew {
	return &GoNew{
		git:    git,
		github: github,
		goH:    goHandler,
		log:    func(...any) {},
	}
}

// SetLog sets the logger function
func (gn *GoNew) SetLog(fn func(...any)) {
	gn.log = fn
}

// Create executes full workflow with remote (or local-only fallback)
func (gn *GoNew) Create(opts NewProjectOptions) (string, error) {
	// 1. Validate inputs
	if err := ValidateRepoName(opts.Name); err != nil {
		return "", err
	}
	if err := ValidateDescription(opts.Description); err != nil {
		return "", err
	}

	if opts.Visibility == "" {
		opts.Visibility = "public"
	}

	// Determine target directory
	targetDir := opts.Directory
	if targetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		targetDir = filepath.Join(cwd, opts.Name)
	}
	// Expand home tilde if present (simple handle)
	if strings.HasPrefix(targetDir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = filepath.Join(home, targetDir[2:])
		}
	}
	targetDir, _ = filepath.Abs(targetDir)

	// 2. Check availability
	// Check if directory exists
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		return "", fmt.Errorf("directory %s already exists", targetDir)
	}

	// Prepare result summary
	var resultSummary string
	isRemote := false

	// Check git config
	userName, err := gn.git.GetConfigUserName()
	if err != nil || userName == "" {
		return "", fmt.Errorf("git user.name not configured. Run: git config --global user.name \"Name\"")
	}
	_, err = gn.git.GetConfigUserEmail()
	if err != nil {
		// Email is not strictly required for license but needed for commit usually
		return "", fmt.Errorf("git user.email not configured. Run: git config --global user.email \"email@example.com\"")
	}

	// 3. Create remote (if not local-only)
	if !opts.LocalOnly {
		// Check if repo exists on GitHub
		// We need username first
		ghUser, err := gn.github.GetCurrentUser()
		if err != nil {
			// Fallback to local only
			gn.log("GitHub unavailable:", err)
			resultSummary = fmt.Sprintf("⚠️ Created: %s [local only] v0.0.1 - %s", opts.Name, gn.github.GetHelpfulErrorMessage(err))
		} else {
			exists, err := gn.github.RepoExists(ghUser, opts.Name)
			if err == nil && exists {
				return "", fmt.Errorf("repository %s/%s already exists on GitHub", ghUser, opts.Name)
			} else if err != nil {
				// Network error or other issue
				gn.log("GitHub check failed:", err)
				resultSummary = fmt.Sprintf("⚠️ Created: %s [local only] v0.0.1 - gh unavailable", opts.Name)
			} else {
				// Create remote
				if err := gn.github.CreateRepo(opts.Name, opts.Description, opts.Visibility); err != nil {
					gn.log("Failed to create remote:", err)
					resultSummary = fmt.Sprintf("⚠️ Created: %s [local only] v0.0.1 - failed to create remote", opts.Name)
				} else {
					isRemote = true
					resultSummary = fmt.Sprintf("✅ Created: %s [local+remote] v0.0.1", opts.Name)
				}
			}
		}
	} else {
		resultSummary = fmt.Sprintf("⚠️ Created: %s [local only] v0.0.1 - run 'gonew add-remote' when ready", opts.Name)
	}

	// 4. Initialize local
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if isRemote {
		// Clone
		ghUser, err := gn.github.GetCurrentUser()
		if err != nil {
			// Should not happen if isRemote is true
			return "", err
		}
		repoURL := fmt.Sprintf("https://github.com/%s/%s.git", ghUser, opts.Name)
		if _, err := RunCommand("git", "clone", repoURL, targetDir); err != nil {
			return "", fmt.Errorf("failed to clone: %w", err)
		}
	} else {
		// Init local
		if err := gn.git.InitRepo(targetDir); err != nil {
			return "", fmt.Errorf("failed to init repo: %w", err)
		}
	}

	// 5. Generate files
	if err := GenerateREADME(opts.Name, opts.Description, targetDir); err != nil {
		return "", err
	}
	if err := GenerateLicense(userName, targetDir); err != nil {
		return "", err
	}
	if err := GenerateGitignore(targetDir); err != nil {
		return "", err
	}
	if err := GenerateHandlerFile(opts.Name, targetDir); err != nil {
		return "", err
	}

	// Go Mod Init
	// Calculate module path
	// Default: github.com/{username}/{repo-name}
	// We use git config user.name if gh user unavailable? No, usually module path uses github handle.
	// If local only, we might not have gh user.
	// Spec: `module github.com/{username}/{repo-name}`
	var ghUser string
	if gn.github != nil {
		ghUser, err = gn.github.GetCurrentUser()
	} else {
		err = fmt.Errorf("GitHub handler is nil")
	}

	if err != nil {
		// Fallback to git config name or just placeholder
		ghUser = strings.ReplaceAll(strings.ToLower(userName), " ", "")
	}
	modulePath := fmt.Sprintf("github.com/%s/%s", ghUser, opts.Name)

	if err := gn.goH.ModInit(modulePath, targetDir); err != nil {
		return "", fmt.Errorf("go mod init failed: %w", err)
	}

	// Change to target dir for git operations
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(targetDir); err != nil {
		return "", err
	}

	// 6. Initial commit
	if err := gn.git.add(); err != nil {
		return "", err
	}
	if _, err := gn.git.commit("Initial commit"); err != nil {
		return "", err
	}

	// 7. Tag creation
	// Use GenerateNextTag (returns v0.0.1 for new repos)
	// Or just force create v0.0.1
	if _, err := gn.git.createTag("v0.0.1"); err != nil {
		return "", err
	}

	// 8. Push
	if isRemote {
		if err := gn.git.pushWithTags("v0.0.1"); err != nil {
			// If push fails, warn but don't fail the whole process
			gn.log("Push failed:", err)
			resultSummary = fmt.Sprintf("⚠️ Created: %s [local only] v0.0.1 - push failed", opts.Name)
		}
	}

	return resultSummary, nil
}

// AddRemote adds GitHub remote to existing local project
func (gn *GoNew) AddRemote(projectPath, visibility string) (string, error) {
	// ... Implement AddRemote logic ...
	// For now, let's implement the basic structure based on spec.

	targetDir := projectPath
	if targetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		targetDir = cwd
	}
	// Expand path...

	targetDir, _ = filepath.Abs(targetDir)

	// Validate project structure
	if _, err := os.Stat(filepath.Join(targetDir, "go.mod")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a Go project (go.mod missing)")
	}
	if _, err := os.Stat(filepath.Join(targetDir, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository")
	}

	// Read description from README.md (assuming first line # Name\n\nDesc)
	// Or just use a default? Spec says "reads description from README.md"
	description := "Go project" // Default
	readmeBytes, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	if err == nil {
		// Try to parse
		lines := strings.Split(string(readmeBytes), "\n")
		// Look for first non-empty line after title?
		for i, line := range lines {
			if strings.HasPrefix(line, "#") {
				// Title
				if i+2 < len(lines) {
					desc := strings.TrimSpace(lines[i+2])
					if desc != "" {
						description = desc
					}
				}
				break
			}
		}
	}

	// Repo name from dir name
	repoName := filepath.Base(targetDir)

	// Check if remote exists locally
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(targetDir); err != nil {
		return "", err
	}

	// Check existing remotes
	remotes, _ := RunCommandSilent("git", "remote")
	if strings.Contains(remotes, "origin") {
		return fmt.Sprintf("Remote 'origin' already configured for %s", repoName), nil
	}

	// Check if repo exists on GitHub
	ghUser, err := gn.github.GetCurrentUser()
	if err != nil {
		return "", fmt.Errorf("GitHub unavailable: %w", err)
	}

	exists, err := gn.github.RepoExists(ghUser, repoName)
	if err == nil && exists {
		return "", fmt.Errorf("repository %s/%s already exists on GitHub", ghUser, repoName)
	}

	// Create remote
	if visibility == "" {
		visibility = "public"
	}
	if err := gn.github.CreateRepo(repoName, description, visibility); err != nil {
		return "", fmt.Errorf("failed to create remote: %w", err)
	}

	// Add remote
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", ghUser, repoName)
	if _, err := RunCommand("git", "remote", "add", "origin", repoURL); err != nil {
		return "", fmt.Errorf("failed to add remote: %w", err)
	}

	// Push
	// We need to push current branch to main
	// And push tags
	if err := gn.git.pushWithTags("v0.0.1"); err != nil {
		// If fails, maybe we need to push plain first?
		// Or maybe v0.0.1 doesn't exist?
		// Try pushing HEAD
		if _, err := RunCommand("git", "push", "-u", "origin", "main"); err != nil {
			return "", fmt.Errorf("failed to push: %w", err)
		}
		// Try pushing tags if any
		RunCommand("git", "push", "--tags")
	}

	return fmt.Sprintf("✅ Remote added: %s/%s", ghUser, repoName), nil
}
