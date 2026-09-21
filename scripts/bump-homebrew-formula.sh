#!/usr/bin/env bash
# Bump Formula/lead.rb in a local checkout of fattman2008/homebrew-tap.
# Usage: scripts/bump-homebrew-formula.sh [/path/to/homebrew-tap]
# Version always comes from internal/version/VERSION (fallback: latest git tag).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAP_DIR="${1:-$HOME/projects/homebrew-tap}"
VERSION=""

if [[ -f "$ROOT/internal/version/VERSION" ]]; then
  VERSION="$(tr -d '[:space:]' < "$ROOT/internal/version/VERSION")"
fi
if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$ROOT" describe --tags --abbrev=0 2>/dev/null || true)"
  VERSION="${VERSION#v}"
fi
if [[ -z "$VERSION" ]]; then
  echo "usage: $0 [tap-dir]" >&2
  echo "  (set internal/version/VERSION first)" >&2
  exit 1
fi

TAG="v${VERSION}"
URL="https://github.com/fattman2008/lead/archive/refs/tags/${TAG}.tar.gz"
SHA="$(curl -fsSL "$URL" | shasum -a 256 | awk '{print $1}')"
FORMULA="${TAP_DIR}/Formula/lead.rb"

if [[ ! -f "$FORMULA" ]]; then
  echo "formula not found: $FORMULA" >&2
  exit 1
fi

# Portable-ish in-place edit for url/sha256/version fields Homebrew cares about.
python3 - "$FORMULA" "$TAG" "$URL" "$SHA" <<'PY'
import re, sys
path, tag, url, sha = sys.argv[1:5]
version = tag.lstrip("v")
text = open(path).read()
text = re.sub(r'url "[^"]*"', f'url "{url}"', text, count=1)
text = re.sub(r'sha256 "[^"]*"', f'sha256 "{sha}"', text, count=1)
# Keep homepage; version comes from the url tag for GitHub archives.
open(path, "w").write(text)
print(f"updated {path}")
print(f"  version {version}")
print(f"  sha256  {sha}")
PY
