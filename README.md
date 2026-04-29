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

Build and run the tool by providing a repository path in the format `owner/repo`:

1. **Build the executable:**
   ```bash
   go build -o gorepohealth
   ```

2. **Run the check:**
   ```bash
   ./gorepohealth google/go-github
   ```

### Example Output
```text
Checking health for: google/go-github

Summary:
- README: Found
- LICENSE: Found
```

## How it works

The tool uses the [google/go-github](https://github.com/google/go-github) library to interact with the GitHub REST API. It authenticates via a Personal Access Token provided through the `GITHUB_TOKEN` environment variable.

It specifically checks for:
- **README**: Using the `GetReadme` endpoint.
- **LICENSE**: Using the `License` endpoint.
- **CI (GitHub Actions)**: Checks for the presence of YAML files in `.github/workflows`.
- **Automated Tests**: Scans workflow files for test commands (e.g., `go test`, `npm test`).

## License

This project is open-source and available under the MIT License.
