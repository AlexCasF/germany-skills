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

## Skill Summary

| Skill | CLI | Short Description |
| --- | --- | --- |
| `abgeordnetenwatch` | `abgeordnetenwatch` | Public transparency data: politician profiles, mandates, side jobs, votes, and profile pages. |
| `bundeshaushalt` | `bundeshaushalt` | German federal budget hierarchy, revenue and spending lines, planned and actual values. |
| `bundesrat-live` | `bundesrat-live` | Bundesrat public website feeds: news, dates, members, plenary summaries, and source pages. |
| `bundestag-live` | `bundestag-live` | Bundestag website feeds: live agenda, members, committees, speeches, articles, and media context. |
| `bundestag-lobbyregister` | `bundestag-lobbyregister` | Bundestag Lobbyregister entries, statements, statistics, and register-source metadata. |
| `dashboard-deutschland` | `dashboard-deutschland` | Dashboard Deutschland indicators, chart-oriented public metrics, sources, and endpoint diagnostics. |
| `destatis` | `destatis` | Official German statistics from Destatis GENESIS-style endpoints. |
| `deutschlandatlas` | `deutschlandatlas` | Deutschlandatlas regional indicators and map-service data. |
| `dip-bundestag` | `dip-bundestag` | Official Bundestag DIP materials: proceedings, printed papers, activities, and document text. |
| `rechtsinformationen-bund` | `rechtsinformationen-bund` | Federal legal information preview API: legislation, case law, and legal document search. |
| `regionalatlas` | `regionalatlas` | Regionalatlas Deutschland indicators across administrative regions and map layers. |
| `tagesschau` | `tagesschau` | Tagesschau public news feeds, search, channels, articles, and source expansion. |

## License

Code and original documentation are Apache-2.0. API-owner reference files and public metadata remain subject to their original publishers' terms.
