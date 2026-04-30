package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Vulnerability represents a security vulnerability found in a package.
type Vulnerability struct {
	Package string `json:"package"`
	Version string `json:"version"`
	ID      string `json:"id"`
}

// OSVQuery represents the request structure for the OSV.dev API.
type OSVQuery struct {
	Version string `json:"version"`
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
}

// OSVResult represents the response structure from the OSV.dev API.
type OSVResult struct {
	Vulns []struct {
		ID string `json:"id"`
	} `json:"vulns"`
}

// CheckOSV queries the OSV.dev API for vulnerabilities in a specific Go package and version.
func CheckOSV(pkg, version string) ([]string, error) {
	query := OSVQuery{Version: version}
	query.Package.Name = pkg
	query.Package.Ecosystem = "Go"

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OSV query: %w", err)
	}

	resp, err := http.Post("https://api.osv.dev/v1/query", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call OSV API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV API returned status: %d", resp.StatusCode)
	}

	var result OSVResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode OSV result: %w", err)
	}

	var ids []string
	for _, v := range result.Vulns {
		ids = append(ids, v.ID)
	}
	return ids, nil
}
