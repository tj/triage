package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v28/github"
	"github.com/tj/go-config"
	"github.com/tj/go-tea"
	"golang.org/x/oauth2"

	"github.com/tj/triage"
)

// defaultPriorities is a set of default user priorities.
var defaultPriorities = []triage.Priority{
	{
		Name:  "Low",
		Label: "Priority: Low",
		Color: "#532BE3",
	},
	{
		Name:  "Medium",
		Label: "Priority: Medium",
		Color: "#532BE3",
	},
	{
		Name:  "High",
		Label: "Priority: High",
		Color: "#532BE3",
	},
}

func main() {
	ctx := context.Background()

	// require GITHUB_TOKEN
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr, "\n  The \033[1mGITHUB_TOKEN\033[m environment variable is required.\n")
		fmt.Fprintf(os.Stderr, "\n  You can generate a personal access token at https://github.com/settings/tokens,\n  then add it to your shell profile, .envrc, or simply `export GITHUB_TOKEN=xxxxxxxx`.\n\n")
		os.Exit(1)
	}

	// http client
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	))
	ctx = triage.NewClientContext(ctx, github.NewClient(httpClient))

	// load config
	var c triage.Config
	err := config.LoadHome(".triage.json", &c)
	if err != nil {
		log.Fatalf("error loading config: %s", err)
	}
	ctx = triage.NewConfigContext(ctx, &c)

	// defaults
	if c.Priorities == nil {
		c.Priorities = defaultPriorities
	}

	// start program
	program := tea.NewProgram(triage.Init, triage.Update, triage.View)
	err = program.Start(ctx)
	if err != nil {
		log.Fatalf("error: %s\r\n", err)
	}

	clear()
}

// clear the screen.
func clear() {
	fmt.Printf("\033[2J\033[3J\033[1;1H")
}
