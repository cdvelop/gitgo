package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tinywasm/devflow"
)

func main() {
	// Subcommands
	addRemoteCmd := flag.NewFlagSet("add-remote", flag.ExitOnError)
	addRemoteVisibility := addRemoteCmd.String("visibility", "public", "Visibility (public/private)")

	// Main command flags
	// We handle main flags manually or via a FlagSet for the root command if no subcommand provided

	// Check for subcommand
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "add-remote":
			addRemoteCmd.Parse(os.Args[2:])
			handleAddRemote(addRemoteCmd.Args(), *addRemoteVisibility)
			return
		}
	}

	// Main command handling (gonew <repo-name> <description>)
	fs := flag.NewFlagSet("gonew", flag.ExitOnError)
	visibilityFlag := fs.String("visibility", "public", "Visibility (public/private)")
	localOnlyFlag := fs.Bool("local-only", false, "Skip remote creation entirely")
	licenseFlag := fs.String("license", "MIT", "License type (default: MIT)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `gonew - Create new Go projects

Usage:
    gonew <repo-name> <description> [flags]
    gonew add-remote <project-path> [flags]

Flags:
    --visibility  public|private (default: public)
    --local-only  Skip remote creation
    --license     License type (default: MIT)

Examples:
    gonew my-project "A sample Go project"
    gonew my-lib "Go library" --visibility=private
    gonew ~/Dev/my-tool "CLI tool" --local-only
    gonew add-remote ./my-project --visibility=public
`)
	}

	// Parse flags for main command
	// Note: os.Args[0] is program name.
	// If args start with -, they are flags. If not, they are positional args, but flags can follow.
	// flag.Parse() expects flags to be first if using default flag set? No, it handles args.
	// But we have positional args first <repo-name> <description>.
	// Go's flag package expects flags BEFORE positional arguments.
	// To support `gonew name desc --flag`, we might need to be careful or just enforce flags first?
	// The spec examples show `gonew my-project "desc" --flags`.
	// Go's flag package stops parsing at the first non-flag argument.

	// Implementation strategy:
	// iterate all args. If it starts with -, treat as flag. If not, treat as positional.
	// This allows flexible flag placement.

	args := os.Args[1:]

	// Reorder args: put flags first, then positional.
	reorderedArgs := []string{}
	purePositional := []string{}

	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if len(arg) > 0 && arg[0] == '-' {
			reorderedArgs = append(reorderedArgs, arg)
			// Check if this flag takes an argument
			// Our flags: -visibility (takes arg), -local-only (bool), -license (takes arg)
			if arg == "--visibility" || arg == "-visibility" ||
			   arg == "--license" || arg == "-license" {
				if i+1 < len(args) {
					reorderedArgs = append(reorderedArgs, args[i+1])
					skipNext = true
				}
			}
		} else {
			purePositional = append(purePositional, arg)
		}
	}

	fs.Parse(reorderedArgs)

	if len(purePositional) < 2 {
		fs.Usage()
		os.Exit(1)
	}

	repoName := purePositional[0]
	description := purePositional[1]

	// Init handlers
	git, err := devflow.NewGit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// We can use NewGitHub, but it might fail if gh not installed.
	// If local-only is set, maybe we don't care?
	// But spec says "NewGitHub verifies gh CLI availability".
	// And "Remote unavailable ... Create local-only".
	// So we should try creating it, if it fails, we note it.

	github, err := devflow.NewGitHub()
	if err != nil {
		// If gh not available, we can still proceed if local-only is requested?
		// Or we can create a dummy GitHub handler that always returns error or just skip it?
		// But NewGoNew requires *GitHub.
		// Let's proceed, but maybe signal that github is broken?
		// Actually, if NewGitHub fails, it means `gh` is not installed.
		// In that case, we should force local-only?
		if !*localOnlyFlag {
			// If user didn't ask for local-only, but we can't use gh, we warn and force local-only?
			// Spec: "Remote unavailable (network, gh CLI issues): Create local-only with helpful message"
			// But that's usually during execution.
			// Here we are at init.
			// We can pass a nil GitHub handler? GoNew struct has *GitHub.
			// Let's pass nil and handle nil in GoNew methods?
			// Or modify GoNew to accept nil?
			// Let's modify GoNew to handle nil github or just ignore the error here and let GoNew handle it?
			// But NewGitHub returns nil struct on error.
			// Let's allow passing nil to NewGoNew and handle it inside.
			github = nil // Explicitly nil
			if !*localOnlyFlag {
				fmt.Println("⚠️  gh CLI not available. Defaulting to local-only mode.")
				*localOnlyFlag = true
			}
		}
	}

	goHandler, err := devflow.NewGo(git)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	orchestrator := devflow.NewGoNew(git, github, goHandler)

	// Create project
	opts := devflow.NewProjectOptions{
		Name:        repoName,
		Description: description,
		Visibility:  *visibilityFlag,
		LocalOnly:   *localOnlyFlag,
		License:     *licenseFlag,
	}

	summary, err := orchestrator.Create(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(summary)
}

func handleAddRemote(args []string, visibility string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: gonew add-remote <project-path> [flags]\n")
		os.Exit(1)
	}

	projectPath := args[0]

	git, err := devflow.NewGit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	github, err := devflow.NewGitHub()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: GitHub CLI (gh) is required for add-remote: %v\n", err)
		os.Exit(1)
	}

	goHandler, err := devflow.NewGo(git)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	orchestrator := devflow.NewGoNew(git, github, goHandler)

	summary, err := orchestrator.AddRemote(projectPath, visibility)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(summary)
}
