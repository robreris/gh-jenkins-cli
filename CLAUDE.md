# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go CLI tool for managing FortinetCloudCSE GitHub repos and Jenkins pipelines. Built with [Cobra](https://github.com/spf13/cobra) for CLI structure, `google/go-github` for GitHub API calls, and raw HTTP for Jenkins API calls.

## Build

```bash
go mod download

# Linux/x86_64 (default target)
GOOS=linux GOARCH=amd64 go build -o gh-jenkins-cli .

# macOS
GOOS=darwin GOARCH=amd64 go build -o gh-jenkins-cli .

# Windows
GOOS=windows GOARCH=amd64 go build -o gh-jenkins-cli.exe .
```

No test suite exists in this project.

## Required Environment Variables

The tool reads credentials exclusively from env vars (no config files):

```bash
export GITHUB_TOKEN=...       # GitHub PAT with repo, admin:repo_hook, delete_repo permissions
export JENKINS_URL=...        # e.g. https://jenkins.example.com:8443
export JENKINS_USER_ID=...
export JENKINS_API_TOKEN=...
```

Use `setenv-template.sh` as a starting point: copy to `setenv.sh`, populate, then `source setenv.sh`.

## Architecture

```
main.go              # Calls cmd.Execute()
cmd/                 # Cobra commands, one file per command
  root.go            # Root command definition and Execute()
  create_repo.go
  create_job.go
  create_project.go  # Composes create_job + create_repo
  delete_repo.go
  delete_job.go
  delete_project.go  # Composes delete_job + delete_repo
  add_collab.go
  rotate_pat.go      # PAT rotation: upserts Jenkins credential + updates GitHub server config
github/
  github.go          # github.Client wrapping go-github; all GitHub API logic
jenkins/
  jenkins_cli.go     # jenkins.APIClient using raw HTTP; CreateJob, DeleteJob, UpsertSecretTextCredential, UpdateGitHubServerCredential
  template-config.xml # Jenkins pipeline XML template; REPO_NAME is replaced at runtime
  Jenkinsfile        # Pushed into created repos; second `when { expression { false } }` block is toggled to `true` when pipeline is enabled
```

### Key design patterns

- **Client construction**: `github.NewClient()` and `jenkins.NewAPIClient()` both read env vars at call time and return struct pointers used by command handlers.
- **`create-project` flow**: Creates Jenkins job first, then GitHub repo. Repo creation (`github.CreateRepo`) does several sequential steps: generate from `UserRepo` template, wait for `main` branch, enable GitHub Pages, create Jenkins webhook, commit updated README + Jenkinsfile, wait for `ci/jenkins/build-status` status check, then apply branch protection.
- **Jenkinsfile toggle**: `UpdateRepoFiles` uses a regex to find `when { expression { false } }` blocks in `github/Jenkinsfile` and replaces only the 2nd occurrence with `true` to enable the pipeline stage in newly created repos.
- **Jenkins auth**: Basic auth via base64-encoded `username:token` header. CSRF crumb is fetched automatically via `getCSRFCrumb()` before any credential or script console POST; a 404/403 on the crumb endpoint is treated as CSRF disabled and silently skipped.
- **Hardcoded org**: GitHub operations target `FortinetCloudCSE` org and use `UserRepo` as the template repository. These are not configurable via flags.

## PAT Rotation (`rotate-pat`)

Automates the manual workflow of replacing an expired GitHub PAT in Jenkins. Classic GitHub PATs cannot be created via API (requires browser UI), so that step remains manual; this command handles everything after.

```bash
./gh-jenkins-cli rotate-pat \
  --credential-id <jenkins-credential-id> \
  --secret <new-pat-value> \
  --expires 2026-11-15
```

**What it does in order:**
1. Upserts a Secret Text credential in the Jenkins System/Global credential store (creates if absent, updates secret value if present)
2. Updates every GitHub Server entry under Manage Jenkins > System > GitHub to use the new credential ID — equivalent to selecting the credential in the dropdown and clicking Apply
3. If `--expires` is set, saves the date to `~/.config/gh-jenkins-cli/pat_expiry` and adds a one-time check snippet to `~/.bashrc` that prints a warning in new terminals when expiry is 14 days or less away

**Flags:**
- `--credential-id` / `-i` (required): Jenkins credential ID to create or update
- `--secret` / `-s` (optional): New PAT value; falls back to `GITHUB_TOKEN` env var if omitted
- `--description` / `-d` (optional, default `"GitHub PAT"`): Credential description in Jenkins
- `--expires` / `-e` (optional): PAT expiration date in `YYYY-MM-DD` format

**Terminal reminder:** Once set up via `--expires`, each new terminal session checks the saved date and prints a warning if expiry is within 14 days (or if already expired). The warning clears automatically after running `rotate-pat` again with a new future `--expires` date.

**Jenkins APIs used:**
- `POST /credentials/store/system/domain/_/createCredentials` — create credential
- `POST /credentials/store/system/domain/_/credential/{id}/config.xml` — update credential
- `POST /scriptText` — Groovy script console to update GitHub plugin server config
