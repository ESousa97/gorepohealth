package health

import (
	"testing"

	"github.com/esousa97/gorepohealth/pkg/security"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		health   RepoHealth
		expected int
	}{
		{
			name: "Perfect Score",
			health: RepoHealth{
				HasReadme:   true,
				HasLicense:  true,
				HasCI:       true,
				HasAutoTest: true,
				Vulnerabilities: []security.Vulnerability{},
			},
			expected: 100,
		},
		{
			name: "Missing Everything",
			health: RepoHealth{
				HasReadme:   false,
				HasLicense:  false,
				HasCI:       false,
				HasAutoTest: false,
				Vulnerabilities: []security.Vulnerability{{ID: "VULN-1"}},
			},
			expected: 0,
		},
		{
			name: "Basic Docs Only",
			health: RepoHealth{
				HasReadme:   true,
				HasLicense:  true,
				HasCI:       false,
				HasAutoTest: false,
				Vulnerabilities: []security.Vulnerability{},
			},
			expected: 70, // 10 (Readme) + 10 (License) + 50 (No Vulns)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.health.CalculateScore()
			if tt.health.Score != tt.expected {
				t.Errorf("expected score %d, got %d", tt.expected, tt.health.Score)
			}
		})
	}
}
