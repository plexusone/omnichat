# Release Notes: v0.3.0

**Release Date:** 2026-03-01

## Summary

This release updates the module path following the GitHub organization rename from `agentplexus` to `plexusone`.

## Breaking Changes

### Module Path Changed

The Go module path has changed from `github.com/agentplexus/omnichat` to `github.com/plexusone/omnichat`.

**Before:**

```go
import "github.com/agentplexus/omnichat/provider"
```

**After:**

```go
import "github.com/plexusone/omnichat/provider"
```

## Upgrade Guide

Update all import statements in your code:

```bash
# Using sed (macOS)
find . -name "*.go" -exec sed -i '' 's|github.com/agentplexus/omnichat|github.com/plexusone/omnichat|g' {} +

# Using sed (Linux)
find . -name "*.go" -exec sed -i 's|github.com/agentplexus/omnichat|github.com/plexusone/omnichat|g' {} +
```

Then update your dependencies:

```bash
go get github.com/plexusone/omnichat@v0.3.0
go mod tidy
```

## Other Changes

### Tests

- Added router tests for `containsURL` and `ProcessWithVoice` covering response modes and URL detection

### Internal

- Fixed `gofmt` formatting in router tests
