# gorepohealth

A Go-based CLI tool to check the health of a GitHub repository by verifying the presence of essential files like `README` and `LICENSE`.

> **Note:** This project is for educational purposes, focusing on Go development and GitHub API integration.

## Prerequisites

- [Go](https://go.dev/doc/install) (1.21 or higher)
- A GitHub Personal Access Token (PAT)

## Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/esousa97/gorepohealth.git
   cd gorepohealth
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set your GitHub Token:**

   **Windows (PowerShell):**
   ```powershell
   $env:GITHUB_TOKEN = "your_personal_access_token"
   ```

   **Linux/macOS:**
   ```bash
   export GITHUB_TOKEN="your_personal_access_token"
   ```

## Usage

### 1. Build the executable
Generate the binary for your operating system:

**Windows:**
```powershell
go build -o dist/gorepohealth.exe ./cmd/gorepohealth/main.go
```

**Linux/macOS:**
```bash
go build -o dist/gorepohealth ./cmd/gorepohealth/main.go
```

### 2. Run analysis

**Analyze a single repository:**
```powershell
./dist/gorepohealth.exe google/go-github
```

**Analyze all repositories of a user and export to CSV:**
```powershell
./dist/gorepohealth.exe ESousa97 --export=resultado.csv
```
*The CSV will be saved in the `outputs/` directory.*


## Testing

You can run the automated test suite (Unit Tests, Build, and Integration Check) using the provided script:

**Windows:**
```powershell
./scripts/test.bat
```

## Project Structure
- `cmd/gorepohealth`: Application entry point.
- `pkg/health`: Core logic for repository analysis and scoring.
- `pkg/security`: Vulnerability scanning via OSV.dev.
- `pkg/report`: Terminal dashboard, CSV, and Markdown generation.
- `scripts/`: Utility scripts for development and testing.
- `outputs/`: Analysis artifacts (CSV and Markdown reports).
- `dist/`: Local build binaries.

## How it works

The tool uses the [google/go-github](https://github.com/google/go-github) library to interact with the GitHub REST API. It authenticates via a Personal Access Token provided through the `GITHUB_TOKEN` environment variable.

It specifically checks for:
- **README**: Using the `GetReadme` endpoint.
- **LICENSE**: Using the `License` endpoint.
- **CI (GitHub Actions)**: Checks for the presence of YAML files in `.github/workflows`.
- **Automated Tests**: Scans workflow files for test commands (e.g., `go test`, `npm test`).
- **Dependency Analysis**: Parses `go.mod` to identify direct dependencies.
- **Security Scanning**: Integrates with the [OSV.dev API](https://osv.dev/) to check for known vulnerabilities in the used libraries.
- **Health Scoring**: Calculates an overall health score (0-100) based on weighted criteria (README, License, CI, Tests, and Security).
- **Markdown Reporting**: Generates a detailed `health_report.md` with the score, analysis breakdown, and suggestions for improvement.
- **Multi-Repo Analysis**: Analyze all public repositories for a specific user.
- **Terminal Dashboard**: Compares health scores across multiple projects in a clean terminal table.
- **CSV Export**: Save analysis results to a CSV file using the `--export` flag.

## License

This project is open-source and available under the MIT License.
