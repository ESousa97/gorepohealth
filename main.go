package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v62/github"
	"golang.org/x/mod/modfile"
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
	fmt.Printf("- CI (GitHub Actions): %s\n", formatResult(health.HasCI))
	fmt.Printf("- Automated Tests: %s\n", formatResult(health.HasAutoTest))

	fmt.Printf("\nDependencies & Security:\n")
	if len(health.Vulnerabilities) == 0 {
		fmt.Println("  [OK] No known vulnerabilities found in direct dependencies.")
	} else {
		for _, v := range health.Vulnerabilities {
			fmt.Printf("  [!] ALERT: %s (version %s) has vulnerability: %s\n", v.Package, v.Version, v.ID)
		}
	}
}

type RepoHealth struct {
	HasReadme       bool
	HasLicense      bool
	HasCI           bool
	HasAutoTest      bool
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Package string
	Version string
	ID      string
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

	// Check for GitHub Actions (CI)
	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, ".github/workflows", nil)
	if err == nil && len(dirContent) > 0 {
		health.HasCI = true
		// Look for tests in workflows
		for _, file := range dirContent {
			if strings.HasSuffix(file.GetName(), ".yml") || strings.HasSuffix(file.GetName(), ".yaml") {
				content, _, _, err := client.Repositories.GetContents(ctx, owner, repo, file.GetPath(), nil)
				if err == nil && content != nil {
					raw, _ := content.GetContent()
					if strings.Contains(raw, "go test") || strings.Contains(raw, "npm test") || strings.Contains(raw, "pytest") || strings.Contains(raw, "test") {
						health.HasAutoTest = true
						break
					}
				}
			}
		}
	}

	// Dependency Analysis (go.mod)
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, "go.mod", nil)
	if err == nil && fileContent != nil {
		raw, _ := fileContent.GetContent()
		f, err := modfile.Parse("go.mod", []byte(raw), nil)
		if err == nil {
			for _, req := range f.Require {
				if req.Indirect {
					continue
				}
				vulns, err := checkOSV(req.Mod.Path, req.Mod.Version)
				if err == nil && len(vulns) > 0 {
					for _, vID := range vulns {
						health.Vulnerabilities = append(health.Vulnerabilities, Vulnerability{
							Package: req.Mod.Path,
							Version: req.Mod.Version,
							ID:      vID,
						})
					}
				}
			}
		}
	}

	return health, nil
}

func checkOSV(pkg, version string) ([]string, error) {
	type osvQuery struct {
		Version string `json:"version"`
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
	}

	query := osvQuery{Version: version}
	query.Package.Name = pkg
	query.Package.Ecosystem = "Go"

	body, _ := json.Marshal(query)
	resp, err := http.Post("https://api.osv.dev/v1/query", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV API error: %d", resp.StatusCode)
	}

	var result struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var ids []string
	for _, v := range result.Vulns {
		ids = append(ids, v.ID)
	}
	return ids, nil
}

func formatResult(found bool) string {
	if found {
		return "Found"
	}
	return "Not Found"
}
