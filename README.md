# Germany Skills

Minimal agent skills and CLIs for read-only research on German public data.

This repo is meant to be installed into an agent runtime, not loaded wholesale into model context. Each installer below discovers every folder under `skills/` and keeps only:

- `skills/<skill>/SKILL.md`
- one runnable CLI flavor for that runtime
- shared wrappers in `bin/`

Skill-specific READMEs, references, tests, and other source variants are intentionally left out of the runtime install.

## One-Line Installs

Choose the runtime your agent can execute.

```bash
curl -fsSL https://raw.githubusercontent.com/AlexCasF/germany-skills/main/scripts/install-python.sh | sh
```

```bash
curl -fsSL https://raw.githubusercontent.com/AlexCasF/germany-skills/main/scripts/install-go.sh | sh
```

```bash
curl -fsSL https://raw.githubusercontent.com/AlexCasF/germany-skills/main/scripts/install-node.sh | sh
```

Defaults:

| Runtime | Installs To | Needs |
| --- | --- | --- |
| Python | `~/.germany-skills/python` | `python3` or `python` |
| Go | `~/.germany-skills/go` | `go` |
| TS/Node.js | `~/.germany-skills/node` | `node` |

After install, add the runtime `bin` folder to `PATH`, for example:

```bash
export PATH="$HOME/.germany-skills/python/bin:$PATH"
dip-bundestag doctor
```

To install somewhere else:

```bash
curl -fsSL https://raw.githubusercontent.com/AlexCasF/germany-skills/main/scripts/install-python.sh | GERMANY_SKILLS_HOME="$PWD/skills" sh
```

To pin a branch, tag, or commit archive:

```bash
curl -fsSL https://raw.githubusercontent.com/AlexCasF/germany-skills/main/scripts/install-node.sh | GERMANY_SKILLS_REF=main sh
```

## Agent Workflow

1. Pick the likely data family.
2. Read only `skills/<skill>/SKILL.md`.
3. Run `<skill> --help`, then subcommand help only if needed.
4. Prefer narrow JSON commands such as `doctor`, `search`, `source`, `text`, or `dossier`.
5. Preserve source URLs, identifiers, timestamps, and warnings returned by the CLI.

## Current Skill CLIs

The installers automatically include every future skill added under `skills/`. The current CLI names are:

`abgeordnetenwatch`, `bundeshaushalt`, `bundesrat-live`, `bundestag-live`, `bundestag-lobbyregister`, `dashboard-deutschland`, `destatis`, `deutschlandatlas`, `dip-bundestag`, `rechtsinformationen-bund`, `regionalatlas`, `tagesschau`.

Some sources need credentials for full use. For example, set `DIP_API_KEY` before using authenticated DIP commands.

## License

Code and original documentation are Apache-2.0. API-owner reference files and public metadata remain subject to their original publishers' terms.
