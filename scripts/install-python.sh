#!/usr/bin/env sh
set -eu

repo_url="${GERMANY_SKILLS_REPO:-https://github.com/AlexCasF/germany-skills}"
ref="${GERMANY_SKILLS_REF:-main}"
dest="${GERMANY_SKILLS_HOME:-$HOME/.germany-skills/python}"

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

if command -v python3 >/dev/null 2>&1; then
  python_cmd=python3
else
  need python
  python_cmd=python
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-python.XXXXXX")
stage=$(mktemp -d "${TMPDIR:-/tmp}/germany-skills-python-stage.XXXXXX")
trap 'rm -rf "$tmp" "$stage"' EXIT

archive_url="$repo_url/archive/$ref.tar.gz"
case "$ref" in
  *.tar.gz|http://*|https://*) archive_url="$ref" ;;
esac

curl -fsSL "$archive_url" | tar -xz -C "$tmp"
src=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n 1)

mkdir -p "$stage/bin"
printf '%s\n' "python" > "$stage/.germany-skills-python"

count=0
for skill_dir in "$src"/skills/*; do
  [ -d "$skill_dir" ] || continue
  [ -f "$skill_dir/SKILL.md" ] || continue
  skill=$(basename "$skill_dir")
  [ -f "$skill_dir/python/$skill.py" ] || {
    echo "Skipping $skill: missing python/$skill.py" >&2
    continue
  }
  count=$((count + 1))
  mkdir -p "$stage/skills/$skill/python"
  cp "$skill_dir/SKILL.md" "$stage/skills/$skill/SKILL.md"
  cp "$skill_dir/python/$skill.py" "$stage/skills/$skill/python/$skill.py"
  chmod +x "$stage/skills/$skill/python/$skill.py"
  cat > "$stage/bin/$skill" <<EOF
#!/usr/bin/env sh
set -eu
script_dir=\$(CDPATH= cd "\$(dirname "\$0")" && pwd)
exec "$python_cmd" "\$script_dir/../skills/$skill/python/$skill.py" "\$@"
EOF
  chmod +x "$stage/bin/$skill"
done

[ "$count" -gt 0 ] || {
  echo "No Python skills found in archive." >&2
  exit 1
}

safe_replace "$dest" ".germany-skills-python"
mkdir -p "$(dirname "$dest")"
mv "$stage" "$dest"
trap 'rm -rf "$tmp"' EXIT

echo "Installed $count Python germany-skills to $dest"
echo "Add to PATH: export PATH=\"$dest/bin:\$PATH\""
