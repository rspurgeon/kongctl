# GitHub Issue Copier

This utility copies all open issues from one GitHub repository to another.
It's particularly useful for copying issues from an upstream repository to
a fork.

## Features

- Fetches all open issues from the source repository
- Creates corresponding issues in the target repository
- Preserves issue titles, bodies, and labels
- Adds a reference to the original issue URL
- Filters out pull requests (only copies actual issues)
- Supports pagination for repositories with many issues
- Includes rate limiting to avoid API throttling
- Provides dry-run mode to preview what will be copied

## Prerequisites

- Go 1.19 or later
- GitHub Personal Access Token with `repo` scope (or at minimum, the
  `public_repo` scope for public repositories)

## Building

```bash
cd scripts
go build -o copy-issues copy-issues.go
```

## Usage

### Environment Variable (Recommended)

```bash
export GITHUB_TOKEN="your_github_token_here"
./copy-issues
```

### Command-line Flag

```bash
./copy-issues -token "your_github_token_here"
```

### Custom Repositories

```bash
./copy-issues \
  -source "Kong/kongctl" \
  -target "your-username/kongctl" \
  -token "your_github_token_here"
```

### Dry Run

Preview what issues would be copied without actually creating them:

```bash
./copy-issues -dry-run
```

## Command-line Options

- `-source` - Source repository in format `owner/repo` (default:
  `Kong/kongctl`)
- `-target` - Target repository in format `owner/repo` (default:
  `rspurgeon/kongctl`)
- `-token` - GitHub personal access token (or set `GITHUB_TOKEN` env var)
- `-dry-run` - Preview issues without creating them

## Output Format

The utility provides progress information:

```
Fetching open issues from Kong/kongctl...
Found 42 open issues

Copying issues to rspurgeon/kongctl...

[1/42] Copying issue #123: Fix authentication bug
  ✓ Created as issue #1: https://github.com/rspurgeon/kongctl/issues/1

[2/42] Copying issue #124: Add new feature
  ✓ Created as issue #2: https://github.com/rspurgeon/kongctl/issues/2

...

Summary:
  Successfully copied: 42
  Failed: 0
  Total: 42
```

## Issue Format

Each copied issue includes:
- Original title (unchanged)
- Reference link to the original issue
- Original issue body
- Original labels (if they exist in the target repo)

Example:

```
_Copied from original issue: https://github.com/Kong/kongctl/issues/123_

---

[Original issue content here]
```

## Rate Limiting

The utility includes a 1-second delay between creating issues to avoid
hitting GitHub's rate limits. For repositories with many issues, the
process may take several minutes.

## Error Handling

- If an individual issue fails to copy, the utility logs the error and
  continues with remaining issues
- A summary at the end shows successful and failed copy operations
- Non-zero exit code if any errors occur during execution

## Security Notes

- Never commit your GitHub token to version control
- Use environment variables or secure credential storage
- The token requires write access to the target repository
- Tokens are sent over HTTPS to GitHub's API

## Example Session

```bash
# Build the utility
cd scripts
go build -o copy-issues copy-issues.go

# Test with dry-run first
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
./copy-issues -dry-run

# If everything looks good, run for real
./copy-issues

# Clean up the token from environment
unset GITHUB_TOKEN
```
