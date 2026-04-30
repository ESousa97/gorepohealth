package health

import (
	"context"
	"strings"

	"github.com/esousa97/gorepohealth/pkg/security"
	"github.com/google/go-github/v62/github"
	"golang.org/x/mod/modfile"
)

// RepoHealth contains all health indicators for a repository.
type RepoHealth struct {
	Name            string
	HasReadme       bool
	HasLicense      bool
	HasCI           bool
	HasAutoTest     bool
	Vulnerabilities []security.Vulnerability
	Score           int
	Suggestions     []string
}

// CalculateScore computes the final health score based on weighted criteria.
func (h *RepoHealth) CalculateScore() {
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

// CheckRepoHealth performs the full health audit on a specific GitHub repository.
func CheckRepoHealth(ctx context.Context, client *github.Client, owner, repo string) (*RepoHealth, error) {
	h := &RepoHealth{Name: repo}

	checkReadme(ctx, client, owner, repo, h)
	checkLicense(ctx, client, owner, repo, h)
	checkCIAndTests(ctx, client, owner, repo, h)
	checkVulnerabilities(ctx, client, owner, repo, h)

	return h, nil
}

func checkReadme(ctx context.Context, client *github.Client, owner, repo string, h *RepoHealth) {
	_, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil {
		h.HasReadme = true
	}
}

func checkLicense(ctx context.Context, client *github.Client, owner, repo string, h *RepoHealth) {
	_, _, err := client.Repositories.License(ctx, owner, repo)
	if err == nil {
		h.HasLicense = true
	}
}

func checkCIAndTests(ctx context.Context, client *github.Client, owner, repo string, h *RepoHealth) {
	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, ".github/workflows", nil)
	if err != nil || len(dirContent) == 0 {
		return
	}

	h.HasCI = true
	for _, file := range dirContent {
		if strings.HasSuffix(file.GetName(), ".yml") || strings.HasSuffix(file.GetName(), ".yaml") {
			content, _, _, err := client.Repositories.GetContents(ctx, owner, repo, file.GetPath(), nil)
			if err == nil && content != nil {
				raw, _ := content.GetContent()
				if strings.Contains(raw, "test") {
					h.HasAutoTest = true
					break
				}
			}
		}
	}
}

func checkVulnerabilities(ctx context.Context, client *github.Client, owner, repo string, h *RepoHealth) {
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, "go.mod", nil)
	if err != nil || fileContent == nil {
		return
	}

	raw, _ := fileContent.GetContent()
	f, err := modfile.Parse("go.mod", []byte(raw), nil)
	if err != nil {
		return
	}

	for _, req := range f.Require {
		if !req.Indirect {
			processDependency(req, h)
		}
	}
}

func processDependency(req *modfile.Require, h *RepoHealth) {
	vulnIDs, err := security.CheckOSV(req.Mod.Path, req.Mod.Version)
	if err == nil && len(vulnIDs) > 0 {
		for _, vID := range vulnIDs {
			h.Vulnerabilities = append(h.Vulnerabilities, security.Vulnerability{
				Package: req.Mod.Path,
				Version: req.Mod.Version,
				ID:      vID,
			})
		}
	}
}
