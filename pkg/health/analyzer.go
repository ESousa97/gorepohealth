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
	HasAutoTest      bool
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
	health := &RepoHealth{Name: repo}

	// 1. Check for README
	_, _, err := client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil {
		health.HasReadme = true
	}

	// 2. Check for License
	_, _, err = client.Repositories.License(ctx, owner, repo)
	if err == nil {
		health.HasLicense = true
	}

	// 3. Check for GitHub Actions (CI)
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

	// 4. Dependency Analysis (go.mod)
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, "go.mod", nil)
	if err == nil && fileContent != nil {
		raw, _ := fileContent.GetContent()
		f, err := modfile.Parse("go.mod", []byte(raw), nil)
		if err == nil {
			for _, req := range f.Require {
				if req.Indirect {
					continue
				}
				vulnIDs, err := security.CheckOSV(req.Mod.Path, req.Mod.Version)
				if err == nil && len(vulnIDs) > 0 {
					for _, vID := range vulnIDs {
						health.Vulnerabilities = append(health.Vulnerabilities, security.Vulnerability{
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
