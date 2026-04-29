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

	health.calculateScore()
	
	fmt.Println("\nSummary:")
	fmt.Printf("- README: %s\n", formatResult(health.HasReadme))
	fmt.Printf("- LICENSE: %s\n", formatResult(health.HasLicense))
	fmt.Printf("- CI (GitHub Actions): %s\n", formatResult(health.HasCI))
	fmt.Printf("- Automated Tests: %s\n", formatResult(health.HasAutoTest))
	fmt.Printf("- Health Score: %d/100\n", health.Score)

	fmt.Printf("\nDependencies & Security:\n")
	if len(health.Vulnerabilities) == 0 {
		fmt.Println("  [OK] No known vulnerabilities found in direct dependencies.")
	} else {
		for _, v := range health.Vulnerabilities {
			fmt.Printf("  [!] ALERT: %s (version %s) has vulnerability: %s\n", v.Package, v.Version, v.ID)
		}
	}

	reportPath := "health_report.md"
	err = health.generateReport(owner, repoName, reportPath)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
	} else {
		fmt.Printf("\nDetailed report generated: %s\n", reportPath)
	}
}

type RepoHealth struct {
	HasReadme       bool
	HasLicense      bool
	HasCI           bool
	HasAutoTest      bool
	Vulnerabilities []Vulnerability
	Score           int
	Suggestions     []string
}

func (h *RepoHealth) calculateScore() {
	h.Score = 0
	h.Suggestions = []string{}

	if h.HasReadme {
		h.Score += 10
	} else {
		h.Suggestions = append(h.Suggestions, "Add a README.md file to document your project.")
	}

	if h.HasLicense {
		h.Score += 10
	} else {
		h.Suggestions = append(h.Suggestions, "Add a LICENSE file to define how others can use your code.")
	}

	if h.HasCI {
		h.Score += 15
	} else {
		h.Suggestions = append(h.Suggestions, "Configure GitHub Actions to automate your build and CI process.")
	}

	if h.HasAutoTest {
		h.Score += 15
	} else {
		h.Suggestions = append(h.Suggestions, "Implement automated tests and ensure they are running in your CI/CD pipeline.")
	}

	if len(h.Vulnerabilities) == 0 {
		h.Score += 50
	} else {
		h.Suggestions = append(h.Suggestions, "Fix identified security vulnerabilities in your dependencies.")
	}
}

func (h *RepoHealth) generateReport(owner, repo, path string) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# Health Report: %s/%s\n\n", owner, repo))
	buf.WriteString(fmt.Sprintf("## Overall Score: **%d/100**\n\n", h.Score))
	
	buf.WriteString("### Analysis Breakdown\n")
	buf.WriteString(fmt.Sprintf("- **README:** %s\n", formatResult(h.HasReadme)))
	buf.WriteString(fmt.Sprintf("- **LICENSE:** %s\n", formatResult(h.HasLicense)))
	buf.WriteString(fmt.Sprintf("- **CI (GitHub Actions):** %s\n", formatResult(h.HasCI)))
	buf.WriteString(fmt.Sprintf("- **Automated Tests:** %s\n", formatResult(h.HasAutoTest)))
	buf.WriteString(fmt.Sprintf("- **Security (Vulnerabilities):** %s\n\n", formatResult(len(h.Vulnerabilities) == 0)))

	if len(h.Vulnerabilities) > 0 {
		buf.WriteString("### Security Alerts\n")
		for _, v := range h.Vulnerabilities {
			buf.WriteString(fmt.Sprintf("- [!] %s (%s): %s\n", v.Package, v.Version, v.ID))
		}
		buf.WriteString("\n")
	}

	buf.WriteString("### Suggestions for Improvement\n")
	if len(h.Suggestions) == 0 {
		buf.WriteString("Great job! Your repository follows all checked health standards.\n")
	} else {
		for _, s := range h.Suggestions {
			buf.WriteString(fmt.Sprintf("- [ ] %s\n", s))
		}
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
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
