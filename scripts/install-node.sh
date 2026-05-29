#!/usr/bin/env sh
set -eu

repo_url="${GERMANY_SKILLS_REPO:-https://github.com/AlexCasF/germany-skills}"
ref="${GERMANY_SKILLS_REF:-main}"
dest="${GERMANY_SKILLS_HOME:-$HOME/.germany-skills/node}"

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
need node

tmp=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-node.XXXXXX")
stage=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-node-stage.XXXXXX")
trap 'rm -rf "$tmp" "$stage"' EXIT

archive_url="$repo_url/archive/$ref.tar.gz"
case "$ref" in
  *.tar.gz|http://*|https://*) archive_url="$ref" ;;
esac

curl -fsSL "$archive_url" | tar -xz -C "$tmp"
src=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n 1)

mkdir -p "$stage/bin"
printf '%s\n' "node" > "$stage/.germany-skills-node"

count=0
for skill_dir in "$src"/skills/*; do
  [ -d "$skill_dir" ] || continue
  [ -f "$skill_dir/SKILL.md" ] || continue
  skill=$(basename "$skill_dir")
  [ -f "$skill_dir/typescript/dist/index.js" ] || {
    echo "Skipping $skill: missing typescript/dist/index.js" >&2
    continue
  }
  count=$((count + 1))
  mkdir -p "$stage/skills/$skill/typescript/dist"
  cp "$skill_dir/SKILL.md" "$stage/skills/$skill/SKILL.md"
  cp "$skill_dir/typescript/dist/index.js" "$stage/skills/$skill/typescript/dist/index.js"
  printf '{ "type": "module", "private": true }\n' > "$stage/skills/$skill/typescript/package.json"
  cat > "$stage/bin/$skill" <<EOF
#!/usr/bin/env sh
set -eu
script_dir=\$(CDPATH= cd "\$(dirname "\$0")" && pwd)
exec node "\$script_dir/../skills/$skill/typescript/dist/index.js" "\$@"
EOF
  chmod +x "$stage/bin/$skill"
done

[ "$count" -gt 0 ] || {
  echo "No Node skills found in archive." >&2
  exit 1
}

safe_replace "$dest" ".germany-skills-node"
mkdir -p "$(dirname "$dest")"
mv "$stage" "$dest"
trap 'rm -rf "$tmp"' EXIT

echo "Installed $count Node germany-skills to $dest"
echo "Add to PATH: export PATH=\"$dest/bin:\$PATH\""
