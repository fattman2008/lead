#!/usr/bin/env bash
# Bump Formula/lead.rb in a local checkout of fattman2008/homebrew-tap.
# Usage: scripts/bump-homebrew-formula.sh [version] [/path/to/homebrew-tap]
set -euo pipefail

VERSION="${1:-}"
TAP_DIR="${2:-$HOME/projects/homebrew-tap}"

if [[ -z "$VERSION" ]]; then
  VERSION="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  VERSION="${VERSION#v}"
fi
if [[ -z "$VERSION" ]]; then
  echo "usage: $0 <version> [tap-dir]" >&2
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
