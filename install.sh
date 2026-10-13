#!/bin/sh
# Install gitix on Linux or macOS:
#   curl -fsSL https://raw.githubusercontent.com/hbeanjarah/gitix/main/install.sh | sh
# Options (environment variables):
#   GITIX_VERSION      tag to install, e.g. v0.1.0 (default: newest release, pre-releases included)
#   GITIX_INSTALL_DIR  where to put the binary (default: /usr/local/bin)
set -eu

repo="hbeanjarah/gitix"
dir="${GITIX_INSTALL_DIR:-/usr/local/bin}"

fail() {
	echo "gitix install: $*" >&2
	exit 1
}

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported system $(uname -s); on Windows use install.ps1" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "unsupported architecture $(uname -m)" ;;
esac

command -v git >/dev/null 2>&1 || echo "Warning: git is not installed; gitix needs git 2.23 or newer." >&2

tag="${GITIX_VERSION:-}"
if [ -z "$tag" ]; then
	tag=$(curl -fsSL "https://api.github.com/repos/$repo/releases?per_page=1" |
		grep '"tag_name"' | head -n 1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
	[ -n "$tag" ] || fail "cannot find the latest release"
fi
version="${tag#v}"
archive="gitix_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading gitix $tag ($os/$arch)..."
curl -fsSL "$base/$archive" -o "$tmp/$archive" || fail "download failed: $base/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || fail "download failed: $base/checksums.txt"

expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
else
	actual=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
fi
[ -n "$expected" ] && [ "$expected" = "$actual" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" gitix

sudo=""
if ! mkdir -p "$dir" 2>/dev/null || [ ! -w "$dir" ]; then
	command -v sudo >/dev/null 2>&1 || fail "$dir is not writable; set GITIX_INSTALL_DIR"
	echo "Installing to $dir needs administrator rights."
	sudo="sudo"
	$sudo mkdir -p "$dir"
fi
$sudo install -m 755 "$tmp/gitix" "$dir/gitix"
# The binary is not signed: make sure macOS does not block it.
if [ "$os" = darwin ]; then
	$sudo xattr -d com.apple.quarantine "$dir/gitix" 2>/dev/null || true
fi

echo "Installed $("$dir/gitix" --version) in $dir"
case ":$PATH:" in
*":$dir:"*) echo "Run: gitix --help" ;;
*) echo "Add $dir to your PATH, then run: gitix --help" ;;
esac
