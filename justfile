tap := env_var_or_default("HOMEBREW_TAP", env_var("HOME") / "projects" / "homebrew-tap")

# List available recipes.
default:
    @just --list

# Run all Go tests.
test:
    go test ./...

# Build bin/pt (version embedded from internal/version/VERSION).
build:
    mkdir -p bin
    go build -o bin/pt ./cmd/pt

# Build and run pt doctor.
doctor: build
    ./bin/pt doctor

# Build and run pt setup.
setup: build
    ./bin/pt setup

# Run tests then doctor.
check: test doctor

# Format Go sources.
fmt:
    go fmt ./...

# Tidy go.mod / go.sum.
tidy:
    go mod tidy

# Create and push git tag v$(VERSION) for the current commit.
tag:
    #!/usr/bin/env bash
    set -euo pipefail
    version="$(tr -d '[:space:]' < internal/version/VERSION)"
    tag="v${version}"
    git tag "$tag"
    git push origin HEAD "$tag"

# Tag/push this repo, then bump the Homebrew formula.
release: tag bump-formula

# Bump Formula/lead.rb in a local homebrew-tap checkout (version from VERSION).
bump-formula tap=tap:
    ./scripts/bump-homebrew-formula.sh {{tap}}
