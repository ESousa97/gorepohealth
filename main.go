package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/mod/modfile"
	"golang.org/x/oauth2"
)

type Vulnerability struct {
	Package string
	Version string
	ID      string
}

type RepoHealth struct {
	Name            string
	HasReadme       bool
	HasLicense      bool
	HasCI           bool
	HasAutoTest      bool
	Vulnerabilities []Vulnerability
	Score           int
	Suggestions     []string
}

func main() {
	exportCSV := flag.String("export", "", "Export results to CSV (e.g., --export=results.csv)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: gorepohealth [options] <owner/repo> or <owner>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	target := args[0]

	// Manual check for the export flag if it was passed after the target
	if *exportCSV == "" {
		for i, arg := range os.Args {
			if strings.HasPrefix(arg, "--export=") {
				*exportCSV = strings.TrimPrefix(arg, "--export=")
			} else if arg == "--export" && i+1 < len(os.Args) {
				*exportCSV = os.Args[i+1]
			}
		}
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var reposToAnalyze []string
	var owner string

	if strings.Contains(target, "/") {
		reposToAnalyze = append(reposToAnalyze, target)
		parts := strings.Split(target, "/")
		owner = parts[0]
	} else {
		owner = target
		fmt.Printf("Fetching all public repositories for user: %s...\n", owner)
		opt := &github.RepositoryListOptions{Type: "public", ListOptions: github.ListOptions{PerPage: 100}}
		repos, _, err := client.Repositories.List(ctx, owner, opt)
		if err != nil {
			fmt.Printf("Error fetching repositories: %v\n", err)
			os.Exit(1)
		}
		for _, r := range repos {
			reposToAnalyze = append(reposToAnalyze, fmt.Sprintf("%s/%s", owner, r.GetName()))
		}
	}

	results := []RepoHealth{}
	for _, repoPath := range reposToAnalyze {
		parts := strings.Split(repoPath, "/")
		repoOwner := parts[0]
		repoName := parts[1]
		fmt.Printf("Analyzing %s...\n", repoPath)
		health, err := checkRepoHealth(ctx, client, repoOwner, repoName)
		if err != nil {
			fmt.Printf("  Error analyzing %s: %v\n", repoName, err)
			continue
		}
		health.Name = repoName
		health.calculateScore()
		results = append(results, *health)
	}

	displayDashboard(results)

	if len(results) > 1 {
		var totalScore int
		for _, r := range results {
			totalScore += r.Score
		}
		average := float64(totalScore) / float64(len(results))
		fmt.Printf("\nPortfolio Health Average: %.2f/100\n", average)
	}

	if *exportCSV != "" {
		err := exportToCSV(results, *exportCSV)
		if err != nil {
			fmt.Printf("Error exporting to CSV: %v\n", err)
		} else {
			fmt.Printf("Results exported to %s\n", *exportCSV)
		}
	}

	if len(reposToAnalyze) == 1 {
		reportPath := "health_report.md"
		err := results[0].generateReport(owner, results[0].Name, reportPath)
		if err != nil {
			fmt.Printf("Error generating report: %v\n", err)
		} else {
			fmt.Printf("\nDetailed report generated: %s\n", reportPath)
		}
	}
}

func (h *RepoHealth) calculateScore() {
	h.Score = 0
	h.Suggestions = []string{}

	if h.HasReadme {
		h.Score += 10
	} else {
		h.Suggestions = append(h.Suggestions, "Add a README.md")
	}

	if h.HasLicense {
		h.Score += 10
	} else {
		h.Suggestions = append(h.Suggestions, "Add a LICENSE")
	}

	if h.HasCI {
		h.Score += 15
	} else {
		h.Suggestions = append(h.Suggestions, "Configure CI")
	}

	if h.HasAutoTest {
		h.Score += 15
	} else {
		h.Suggestions = append(h.Suggestions, "Implement Tests")
	}

	if len(h.Vulnerabilities) == 0 {
		h.Score += 50
	} else {
		h.Suggestions = append(h.Suggestions, "Fix Vulnerabilities")
	}
}

func displayDashboard(results []RepoHealth) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Repository", "Readme", "License", "CI", "Tests", "Security", "Score")

	for _, r := range results {
		secStatus := "OK"
		if len(r.Vulnerabilities) > 0 {
			secStatus = "VULN"
		}

		table.Append(
			r.Name,
			formatFound(r.HasReadme),
			formatFound(r.HasLicense),
			formatFound(r.HasCI),
			formatFound(r.HasAutoTest),
			secStatus,
			strconv.Itoa(r.Score),
		)
	}
	fmt.Println("\n--- Health Dashboard ---")
	table.Render()
}

func formatFound(found bool) string {
	if found {
		return "YES"
	}
	return "NO"
}

func exportToCSV(results []RepoHealth, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Repository", "Readme", "License", "CI", "Tests", "Security_Vulns", "Score"})
	for _, r := range results {
		writer.Write([]string{
			r.Name,
			strconv.FormatBool(r.HasReadme),
			strconv.FormatBool(r.HasLicense),
			strconv.FormatBool(r.HasCI),
			strconv.FormatBool(r.HasAutoTest),
			strconv.Itoa(len(r.Vulnerabilities)),
			strconv.Itoa(r.Score),
		})
	}
	return nil
}

func (h *RepoHealth) generateReport(owner, repo, path string) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# Health Report: %s/%s\n\n", owner, repo))
	buf.WriteString(fmt.Sprintf("## Overall Score: **%d/100**\n\n", h.Score))
	buf.WriteString("### Analysis Breakdown\n")
	buf.WriteString(fmt.Sprintf("- **README:** %v\n", h.HasReadme))
	buf.WriteString(fmt.Sprintf("- **LICENSE:** %v\n", h.HasLicense))
	buf.WriteString(fmt.Sprintf("- **CI (GitHub Actions):** %v\n", h.HasCI))
	buf.WriteString(fmt.Sprintf("- **Automated Tests:** %v\n", h.HasAutoTest))
	buf.WriteString(fmt.Sprintf("- **Security (Vulnerabilities):** %v\n\n", len(h.Vulnerabilities) == 0))

	if len(h.Suggestions) > 0 {
		buf.WriteString("### Suggestions\n")
		for _, s := range h.Suggestions {
			buf.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func checkRepoHealth(ctx context.Context, client *github.Client, owner, repo string) (*RepoHealth, error) {
	health := &RepoHealth{}
	_, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil {
		health.HasReadme = true
	}
	_, _, err = client.Repositories.License(ctx, owner, repo)
	if err == nil {
		health.HasLicense = true
	}
	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, ".github/workflows", nil)
	if err == nil && len(dirContent) > 0 {
		health.HasCI = true
		for _, file := range dirContent {
			if strings.HasSuffix(file.GetName(), ".yml") || strings.HasSuffix(file.GetName(), ".yaml") {
				content, _, _, err := client.Repositories.GetContents(ctx, owner, repo, file.GetPath(), nil)
				if err == nil && content != nil {
					raw, _ := content.GetContent()
					if strings.Contains(raw, "test") {
						health.HasAutoTest = true
						break
					}
				}
			}
		}
	}
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
	var result struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	var ids []string
	for _, v := range result.Vulns {
		ids = append(ids, v.ID)
	}
	return ids, nil
}
