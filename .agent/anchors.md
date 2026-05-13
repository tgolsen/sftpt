# Anchors

## Project: sftpt (CLI SFTP Utility)

**Purpose:** Command-line SFTP utility for Mac, designed for shell script integration

## CLI Development Patterns

### Testing SFTP Operations
```bash
# Local SFTP test server setup (when needed):
# TEST_SFTP_HOST=localhost
# TEST_SFTP_USER=testuser
# TEST_SFTP_PATH=/tmp/sftpt-test

# CLI testing commands (TBD based on chosen language):
# [build command] && ./sftpt --version
# ./sftpt list testuser@localhost:/tmp/test/
```

### CLI Design Requirements
- **Exit codes**: 0=success, non-zero=error (for script integration)
- **Output**: Clean, parseable output for shell scripts
- **Authentication**: SSH keys preferred over passwords for automation
- **Error handling**: Clear messages, graceful failures

## Development Workflow Permissions

### Authorized Actions (No Permission Required)
- CLI tool development and testing
- Branch operations: create, commit, push to feature branches
- Create PRs for CLI features
- Test SFTP operations with local/test servers
- Run authorized CLI tools (gh, acli)

### EXPLICITLY NOT AUTHORIZED
- Merging PRs without explicit permission
- Publishing CLI releases without permission

## Security File Patterns
```bash
# NEVER commit these patterns:
.env*
**/id_rsa*
**/credentials*
**/config.local.*

# Always use specific staging:
git add src/main.go          # ✓ CORRECT
git add .                    # ✗ FORBIDDEN
```

## Git Workflow (Security Required)
```bash
# Correct staging workflow:
git status                   # Review what changed
git diff                     # Review changes
git add src/specific-file    # Stage specific files only
git diff --staged            # Review staged changes
git commit -m "feat: add list command"

# Milestone tagging:
git tag -a v0.1.0 -m "Milestone 1: Basic SFTP operations"
git push origin main --tags
```

## Add sections below ONLY when Claude Code makes repeated mistakes with CLI patterns

