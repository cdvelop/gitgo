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
	addRemoteOwner := addRemoteCmd.String("owner", "", "GitHub owner/organization (default: auto-detected)")
	addRemoteVisibility := addRemoteCmd.String("visibility", "public", "Visibility (public/private)")

	// Main command flags
	// We handle main flags manually or via a FlagSet for the root command if no subcommand provided

	// Check for subcommand
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "add-remote":
			addRemoteCmd.Parse(os.Args[2:])
			handleAddRemote(addRemoteCmd.Args(), *addRemoteVisibility, *addRemoteOwner)
			return
		}
	}

	// Main command handling (gonew <repo-name> <description>)
	fs := flag.NewFlagSet("gonew", flag.ExitOnError)
	ownerFlag := fs.String("owner", "", "GitHub owner/organization (default: auto-detected from gh or git)")
	visibilityFlag := fs.String("visibility", "public", "Visibility (public/private)")
	localOnlyFlag := fs.Bool("local-only", false, "Skip remote creation entirely")
	licenseFlag := fs.String("license", "MIT", "License type (default: MIT)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `gonew - Create new Go projects

Usage:
    gonew <repo-name> <description> [flags]
    gonew add-remote <project-path> [flags]

Flags:
    -owner       GitHub owner/organization (default: auto-detected)
    -visibility  public|private (default: public)
    -local-only  Skip remote creation
    -license     License type (default: MIT)

Examples:
    gonew my-project "A sample Go project"
    gonew my-lib "Go library" -owner=cdvelop
    gonew my-tool "CLI tool" -owner=veltylabs -visibility=private
    gonew ~/Dev/my-tool "CLI tool" -local-only
    gonew add-remote ./my-project -owner=tinywasm -visibility=public
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
			if arg == "--owner" || arg == "-owner" ||
				arg == "--visibility" || arg == "-visibility" ||
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

	// Logger for all operations
	log := func(args ...any) { fmt.Println(args...) }

	// Use Future for GitHub initialization
	var githubFuture *devflow.Future
	if !*localOnlyFlag {
		githubFuture = devflow.NewFuture(func() (any, error) {
			return devflow.NewGitHub(log)
		})
	}

	goHandler, err := devflow.NewGo(git)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	orchestrator := devflow.NewGoNew(git, githubFuture, goHandler)

	// Create project
	opts := devflow.NewProjectOptions{
		Name:        repoName,
		Description: description,
		Owner:       *ownerFlag,
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

func handleAddRemote(args []string, visibility, owner string) {
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

	log := func(args ...any) { fmt.Println(args...) }

	githubFuture := devflow.NewFuture(func() (any, error) {
		return devflow.NewGitHub(log)
	})

	goHandler, err := devflow.NewGo(git)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	orchestrator := devflow.NewGoNew(git, githubFuture, goHandler)

	summary, err := orchestrator.AddRemote(projectPath, visibility, owner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(summary)
}
