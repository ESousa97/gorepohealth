package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/esousa97/gorepohealth/pkg/health"
	"github.com/olekukonko/tablewriter"
)

// GenerateMarkdown creates a detailed health report in Markdown format.
func GenerateMarkdown(h *health.RepoHealth, owner, path string) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# Health Report: %s/%s\n\n", owner, h.Name))
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

// DisplayDashboard renders a terminal table with all analysis results.
func DisplayDashboard(results []health.RepoHealth) {
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

// ExportToCSV saves the analysis results to a CSV file.
func ExportToCSV(results []health.RepoHealth, path string) error {
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

func formatFound(found bool) string {
	if found {
		return "YES"
	}
	return "NO"
}
