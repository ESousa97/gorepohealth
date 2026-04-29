package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gorepohealth <owner/repo>")
		os.Exit(1)
	}

	repoPath := os.Args[1]
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		fmt.Println("Invalid repository format. Use owner/repo")
		os.Exit(1)
	}

	owner := parts[0]
	repoName := parts[1]

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	fmt.Printf("Checking health for: %s/%s\n", owner, repoName)
	
	health, err := checkRepoHealth(ctx, client, owner, repoName)
	if err != nil {
		fmt.Printf("Error checking repository: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSummary:")
	fmt.Printf("- README: %s\n", formatResult(health.HasReadme))
	fmt.Printf("- LICENSE: %s\n", formatResult(health.HasLicense))
}

type RepoHealth struct {
	HasReadme  bool
	HasLicense bool
}

func checkRepoHealth(ctx context.Context, client *github.Client, owner, repo string) (*RepoHealth, error) {
	health := &RepoHealth{}

	// Check for README
	_, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil {
		health.HasReadme = true
	} else if githubErr, ok := err.(*github.ErrorResponse); ok && githubErr.Response.StatusCode != 404 {
		return nil, fmt.Errorf("failed to check README: %w", err)
	}

	// Check for License
	_, _, err = client.Repositories.License(ctx, owner, repo)
	if err == nil {
		health.HasLicense = true
	} else if githubErr, ok := err.(*github.ErrorResponse); ok && githubErr.Response.StatusCode != 404 {
		return nil, fmt.Errorf("failed to check LICENSE: %w", err)
	}

	return health, nil
}

func formatResult(found bool) string {
	if found {
		return "Found"
	}
	return "Not Found"
}
