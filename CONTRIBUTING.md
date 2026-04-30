# Contributing to GoRepoHealth

Thanks for your interest in improving GoRepoHealth! This document explains how to set up a development environment, run the test suite, and submit changes for review.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Running Tests](#running-tests)
- [Linting and Formatting](#linting-and-formatting)
- [Commit Conventions](#commit-conventions)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)
- [Adding a New Health Criterion](#adding-a-new-health-criterion)
- [Adding a New Output Format](#adding-a-new-output-format)
- [Project Layout Reminder](#project-layout-reminder)

## Code of Conduct

Be respectful, be constructive, assume good faith. Disagreement is welcome — disrespect is not.

## Getting Started

### Prerequisites

- **Go** >= 1.26.0
- **Git**
- A **GitHub Personal Access Token** with `public_repo` scope for integration smoke tests

### Fork and Clone

```bash
git clone https://github.com/<your-username>/gorepohealth.git
cd gorepohealth
git remote add upstream https://github.com/esousa97/gorepohealth.git

go mod download
```

### Build the Binary

```bash
go build -o dist/gorepohealth ./cmd/gorepohealth/main.go
./dist/gorepohealth --help
```

## Development Workflow

1. **Sync with upstream** before starting any work:

   ```bash
   git fetch upstream
   git checkout master
   git rebase upstream/master
   ```

2. **Create a feature branch**:

   ```bash
   git checkout -b feat/concurrent-multi-repo
   ```

3. **Make focused changes**. One PR = one concern. Refactors that are not strictly required for the change you are making belong in a separate PR.

4. **Run the full test suite** locally before pushing (see [Running Tests](#running-tests)).

5. **Push and open a PR** against `master`.

## Running Tests

### Unit Tests

```bash
go test ./...
```

With coverage:

```bash
go test -cover ./...
```

Verbose output (useful when investigating a single failure):

```bash
go test -v ./pkg/health/
```

### Race Detector

CI runs with the race detector enabled. Match it locally before pushing concurrency-touching changes:

```bash
go test -race -count=1 ./...
```

### Build Verification

CI builds for three OS/arch pairs. Reproduce locally:

```bash
GOOS=linux   GOARCH=amd64 go build -o /dev/null ./cmd/gorepohealth/main.go
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/gorepohealth/main.go
GOOS=darwin  GOARCH=amd64 go build -o /dev/null ./cmd/gorepohealth/main.go
```

### Integration Smoke Test

The repo ships [scripts/test.bat](scripts/test.bat), which:

1. Runs `go test ./...`
2. Builds the binary into `dist/`
3. (If `GITHUB_TOKEN` is set) runs the binary against `google/go-github`

```powershell
$env:GITHUB_TOKEN = 'ghp_...'
./scripts/test.bat
```

A POSIX equivalent is welcome as a contribution.

## Linting and Formatting

### `gofmt` / `goimports`

All Go code must be `gofmt`-clean. Most editors handle this automatically. To enforce manually:

```bash
gofmt -l -w .
```

### `go vet`

```bash
go vet ./...
```

### `golangci-lint` (optional but encouraged)

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run --verbose
```

## Commit Conventions

Use concise, imperative commit messages:

```text
Add concurrent multi-repo analysis with rate-limit awareness
Fix CSV export when filename contains forward slashes
Refactor osv.go to accept ecosystem as a parameter
```

Prefix with a type when the change is narrow:

- `feat:` new feature
- `fix:` bug fix
- `refactor:` non-behavior change
- `docs:` documentation only
- `test:` test changes only
- `ci:` CI/build changes
- `chore:` housekeeping (dependencies, tooling)

The first line should fit in **72 characters**. Use the body for the *why*, not the *what* — the diff is the *what*.

> **Do not** add `Co-Authored-By: Claude` trailers to commits in this repository.

## Pull Request Process

1. **Title**: short, imperative, under 70 characters. Same conventions as commits.
2. **Body**: a `## Summary` of the change in 1–3 bullets, plus a `## Test plan` checklist of how you verified it.
3. **CI must pass** before review: tests, race detector, and the full build matrix.
4. **Keep PRs small**. < 400 lines of diff is the sweet spot. Larger changes should be split.
5. **Update documentation** in the same PR — `README.md`, `docs/ARCHITECTURE.md`, and inline godoc comments where applicable.
6. **One logical change per PR**. If you also fixed a typo, split it.

### Review Checklist

Before requesting review, confirm:

- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes
- [ ] `gofmt`-clean
- [ ] `go vet ./...` clean
- [ ] Build succeeds on all three CI targets
- [ ] New behavior is covered by a test (or documented as deliberately untested)
- [ ] README / ARCHITECTURE updated if the surface changed

## Reporting Issues

When opening an issue, include:

- Go version (`go version`)
- OS and arch
- The exact command you ran
- The full output (or stack trace)
- What you expected vs. what happened

For security-sensitive bugs (e.g. token leakage in logs), do **not** open a public issue — email the author at the address in the [README footer](README.md#author) instead.

## Adding a New Health Criterion

The audit logic lives in [pkg/health/analyzer.go](pkg/health/analyzer.go). To add a new criterion:

1. **Extend the struct**:

   ```go
   type RepoHealth struct {
       // existing fields...
       HasSecurityPolicy bool
   }
   ```

2. **Populate it in `CheckRepoHealth`**:

   ```go
   _, _, _, err = client.Repositories.GetContents(ctx, owner, repo, "SECURITY.md", nil)
   if err == nil {
       health.HasSecurityPolicy = true
   }
   ```

3. **Score it in `CalculateScore`**. Decide on a weight that does not break the 0–100 invariant:

   ```go
   if h.HasSecurityPolicy {
       h.Score += 5
   } else {
       h.Suggestions = append(h.Suggestions, "Add a SECURITY.md")
   }
   ```

   > If you push past 100, **reduce other weights proportionally** — the report layer assumes a 0–100 range.

4. **Update the report layer** in [pkg/report/generator.go](pkg/report/generator.go):
   - Add a column to `DisplayDashboard` and `ExportToCSV`
   - Add a bullet line to `GenerateMarkdown`

5. **Write a test** in [pkg/health/analyzer_test.go](pkg/health/analyzer_test.go):

   ```go
   {
       name: "Has Security Policy",
       health: RepoHealth{
           HasReadme: true, HasLicense: true, HasCI: true, HasAutoTest: true,
           HasSecurityPolicy: true,
           Vulnerabilities:   []security.Vulnerability{},
       },
       expected: 100, // adjust per the new weight scheme
   },
   ```

6. **Update documentation**:
   - The `## Health Scoring` table in `README.md`
   - The `## Checks Performed` section in `README.md`
   - The `## Scoring Algorithm` section in `docs/ARCHITECTURE.md`

## Adding a New Output Format

To add an HTML or JSON sink:

1. Add a function with the same input shape to [pkg/report/generator.go](pkg/report/generator.go):

   ```go
   func ExportToJSON(results []health.RepoHealth, path string) error { ... }
   ```

2. Wire a flag in [cmd/gorepohealth/main.go](cmd/gorepohealth/main.go):

   ```go
   exportJSON := flag.String("export-json", "", "Export results to JSON")
   ```

3. Honor the `outputs/` convention for bare filenames (look at `ExportToCSV` for the existing pattern).

4. Add a row to the `## Output Formats` section of `README.md`.

## Project Layout Reminder

```text
gorepohealth/
├── cmd/gorepohealth/main.go     # entry point
├── pkg/
│   ├── health/                  # audit logic + scoring
│   ├── security/                # OSV.dev client
│   └── report/                  # dashboard / CSV / markdown sinks
├── scripts/test.bat             # Windows smoke-test
├── .github/
│   ├── workflows/ci.yml         # CI pipeline
│   └── dependabot.yml           # weekly dep updates
├── docs/ARCHITECTURE.md         # technical deep-dive
└── README.md
```

Thanks again for contributing!
