package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/esousa97/gorepohealth/pkg/health"
	"github.com/esousa97/gorepohealth/pkg/report"
	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

func main() {
	exportCSV, target := parseFlags()
	ctx, client := initGitHubClient()

	reposToAnalyze, owner := fetchRepositories(ctx, client, target)
	results := analyzeRepositories(ctx, client, reposToAnalyze)

	generateReports(results, exportCSV, owner)
}

func parseFlags() (string, string) {
	exportCSV := flag.String("export", "", "Export results to CSV (e.g., --export=results.csv)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: gorepohealth [options] <owner/repo> or <owner>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	target := args[0]

	// Check for the export flag if it was passed after the target (manual fallback)
	if *exportCSV == "" {
		for i, arg := range os.Args {
			if strings.HasPrefix(arg, "--export=") {
				*exportCSV = strings.TrimPrefix(arg, "--export=")
			} else if arg == "--export" && i+1 < len(os.Args) {
				*exportCSV = os.Args[i+1]
			}
		}
	}
	return *exportCSV, target
}

func initGitHubClient() (context.Context, *github.Client) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return ctx, github.NewClient(tc)
}

func fetchRepositories(ctx context.Context, client *github.Client, target string) ([]string, string) {
	var reposToAnalyze []string
	var owner string

	if strings.Contains(target, "/") {
		reposToAnalyze = append(reposToAnalyze, target)
		parts := strings.Split(target, "/")
		owner = parts[0]
	} else {
		owner = target
		fmt.Printf("Fetching public repositories for user: %s...\n", owner)
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
	return reposToAnalyze, owner
}

func analyzeRepositories(ctx context.Context, client *github.Client, reposToAnalyze []string) []health.RepoHealth {
	results := []health.RepoHealth{}
	for _, repoPath := range reposToAnalyze {
		parts := strings.Split(repoPath, "/")
		repoOwner := parts[0]
		repoName := parts[1]
		fmt.Printf("Analyzing %s...\n", repoPath)

		res, err := health.CheckRepoHealth(ctx, client, repoOwner, repoName)
		if err != nil {
			fmt.Printf("  Error analyzing %s: %v\n", repoName, err)
			continue
		}
		res.CalculateScore()
		results = append(results, *res)
	}
	return results
}

func generateReports(results []health.RepoHealth, exportCSV, owner string) {
	report.DisplayDashboard(results)
	displayAverageScore(results)

	// Ensure outputs directory exists
	os.MkdirAll("outputs", 0755)

	handleCSVExport(results, exportCSV)
	handleMarkdownReport(results, owner)
}

func displayAverageScore(results []health.RepoHealth) {
	if len(results) > 1 {
		var totalScore int
		for _, r := range results {
			totalScore += r.Score
		}
		average := float64(totalScore) / float64(len(results))
		fmt.Printf("\nPortfolio Health Average: %.2f/100\n", average)
	}
}

func handleCSVExport(results []health.RepoHealth, exportCSV string) {
	if exportCSV == "" {
		return
	}
	csvPath := exportCSV
	if !strings.Contains(csvPath, "/") && !strings.Contains(csvPath, "\\") {
		csvPath = "outputs/" + csvPath
	}
	err := report.ExportToCSV(results, csvPath)
	if err != nil {
		fmt.Printf("Error exporting to CSV: %v\n", err)
	} else {
		fmt.Printf("Results exported to %s\n", csvPath)
	}
}

func handleMarkdownReport(results []health.RepoHealth, owner string) {
	if len(results) == 1 {
		reportPath := "outputs/health_report.md"
		err := report.GenerateMarkdown(&results[0], owner, reportPath)
		if err != nil {
			fmt.Printf("Error generating report: %v\n", err)
		} else {
			fmt.Printf("\nDetailed report generated: %s\n", reportPath)
		}
	}
}
