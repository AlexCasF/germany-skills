#!/usr/bin/env sh
set -eu

repo_url="${GERMANY_SKILLS_REPO:-https://github.com/AlexCasF/germany-skills}"
ref="${GERMANY_SKILLS_REF:-main}"
dest="${GERMANY_SKILLS_HOME:-$HOME/.germany-skills/go}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

safe_replace() {
  target=$1
  marker=$2
  case "$target" in
    ""|"/"|".")
      echo "Refusing unsafe install target: $target" >&2
      exit 1
      ;;
  esac
  if [ -e "$target" ] && [ ! -f "$target/$marker" ] && [ "${GERMANY_SKILLS_OVERWRITE:-0}" != "1" ]; then
    echo "Refusing to replace $target because it was not created by this installer." >&2
    echo "Set GERMANY_SKILLS_HOME to a new directory or GERMANY_SKILLS_OVERWRITE=1." >&2
    exit 1
  fi
  rm -rf "$target"
}

need curl
need tar
need go

tmp=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-go.XXXXXX")
stage=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-go-stage.XXXXXX")
trap 'rm -rf "$tmp" "$stage"' EXIT

archive_url="$repo_url/archive/$ref.tar.gz"
case "$ref" in
  *.tar.gz|http://*|https://*) archive_url="$ref" ;;
esac

curl -fsSL "$archive_url" | tar -xz -C "$tmp"
src=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n 1)
goexe=$(go env GOEXE)

mkdir -p "$stage/bin"
printf '%s\n' "go" > "$stage/.germany-skills-go"

count=0
for skill_dir in "$src"/skills/*; do
  [ -d "$skill_dir" ] || continue
  [ -f "$skill_dir/SKILL.md" ] || continue
  skill=$(basename "$skill_dir")
  [ -d "$skill_dir/go" ] || {
    echo "Skipping $skill: missing go/" >&2
    continue
  }
  count=$((count + 1))
  mkdir -p "$stage/skills/$skill"
  cp "$skill_dir/SKILL.md" "$stage/skills/$skill/SKILL.md"
  (cd "$skill_dir/go" && go build -o "$stage/bin/$skill$goexe" .)
done

[ "$count" -gt 0 ] || {
  echo "No Go skills found in archive." >&2
  exit 1
}

safe_replace "$dest" ".germany-skills-go"
mkdir -p "$(dirname "$dest")"
mv "$stage" "$dest"
trap 'rm -rf "$tmp"' EXIT

echo "Installed $count Go germany-skills to $dest"
echo "Add to PATH: export PATH=\"$dest/bin:\$PATH\""
