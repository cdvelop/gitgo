package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tinywasm/devflow"
)

func main() {
	fs := flag.NewFlagSet("gotest", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Silence default flag errors

	showHelp := func() {
		fmt.Println("Usage: gotest")
		fmt.Println("Automated flow. No arguments needed.")
	}

	err := fs.Parse(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			showHelp()
			os.Exit(0)
		}
		// Minimal error for flags like -v
		fmt.Println("gotest: no arguments needed.")
		os.Exit(1)
	}

	// Check handling for help args
	if len(fs.Args()) > 0 {
		arg := fs.Args()[0]
		if arg == "?" || arg == "help" {
			showHelp()
			os.Exit(0)
		}
		fmt.Printf("gotest: unexpected %q. No arguments needed.\n", arg)
		os.Exit(1)
	}

	git, err := devflow.NewGit()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	goHandler, err := devflow.NewGo(git)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	summary, err := goHandler.Test()
	if err != nil {
		fmt.Println("Tests failed:", err)
		os.Exit(1)
	}

	fmt.Println(summary)
}
