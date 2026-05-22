# Germany Skills

Agent-friendly command-line tools and `SKILL.md` instructions for researching German public data.

This repository packages a set of read-only research skills around German parliamentary, legal, statistical, budget, transparency, regional, and news data sources. Each skill is designed for agents that can read files and run shell commands: the agent first reads the relevant `SKILL.md`, inspects the CLI help only when needed, and then runs narrow JSON-producing commands with source links.

The core idea is progressive disclosure. Instead of putting every OpenAPI schema into the model context at startup, each API family lives in its own folder with concise guidance, reference notes, and Go/Python/TypeScript CLI implementations.

## Included Skills

| Skill folder | Main CLI | Data focus |
| --- | --- | --- |
| `abgeordnetenwatch` | `abgeordnetenwatchctl` | Politician profiles, mandates, side jobs, voting and transparency data from abgeordnetenwatch.de. |
| `bundeshaushalt` | `bundeshaushaltctl` | German federal budget hierarchy, revenue and spending lines. |
| `bundesrat-live` | `bundesratctl` | Bundesrat public website data, dates, news, members, and plenary context. |
| `bundestag-live` | `bundestagctl` | Bundestag live/public website feeds, agenda, members, and committee context. |
| `bundestag-lobbyregister` | `lobbyregisterctl` | Bundestag Lobbyregister entries and interest-representation metadata. |
| `dashboard-deutschland` | `dashboardctl` | Dashboard Deutschland indicators and chart-oriented public metrics. |
| `destatis` | `destatisctl` | Official German statistics from Destatis GENESIS-style endpoints. |
| `deutschlandatlas` | `deutschlandatlasctl` | Deutschlandatlas regional indicators and map-service data. |
| `dip-bundestag` | `dipctl` | Official Bundestag DIP materials, printed papers, proceedings, activities, and plenary protocol text. |
| `rechtsinformationen-bund` | `rechtsinformationenctl` | Official German federal legal information trial API: legislation, case law, and legal documents. |
| `regionalatlas` | `regionalatlasctl` | Regionalatlas Deutschland indicators across administrative regions. |
| `tagesschau` | `tagesschauctl` | Tagesschau public news feeds, search, channels, and article expansion. |

## Repository Layout

Each skill folder follows the same general shape:

```text
<skill>/
  SKILL.md                 agent-facing usage guidance
  README.md                human-facing tool notes, where available
  references/              OpenAPI files, research notes, rate-limit notes
  go/v2/                   standalone Go CLI source
  python/                  Python CLI implementation
  typescript/              TypeScript/Node.js CLI implementation
  bin/                     locally built Windows binaries, where available
  tests/                   manual test plan and observed results
```

## Quick Start

Clone the repository:

```bash
git clone https://github.com/AlexCasF/germany-skills.git
cd germany-skills
```

Run a Go CLI directly from source:

```bash
cd dip-bundestag/go/v2 && go run . doctor
```

Run the Python flavor:

```bash
python dip-bundestag/python/dipctl.py doctor
```

Build and run the TypeScript/Node.js flavor:

```bash
npm --prefix dip-bundestag/typescript ci && npm --prefix dip-bundestag/typescript run build && node dip-bundestag/typescript/dist/index.js doctor
```

Run a prebuilt Windows binary where available:

```powershell
.\dip-bundestag\bin\dipctl-2.0.exe doctor
```

## For Agents

Start with [AGENTS.md](AGENTS.md). It explains how Codex, Claude Code, Hermes/OpenClaw/Pi/Picoclaw-style computer agents, and ADK-style agents can discover and use the skill folders without loading the whole repository into context.

The safest default workflow is:

1. Read the target skill's `SKILL.md`.
2. Run `<cli> --help`.
3. Run `<cli> <group> --help` only when needed.
4. Prefer `doctor`, `search`, `source`, `text`, and `dossier` helpers before broad endpoint calls.
5. Preserve returned source URLs in final answers.

## API Keys And Fair Use

Most data sources are unauthenticated public APIs. Some endpoints require credentials or have documented fair-use expectations. Check each skill's `references/rate-limits-and-terms.md` before high-volume use.

For DIP Bundestag, set `DIP_API_KEY` when using `dipctl`:

```bash
export DIP_API_KEY="your-key"
```

On PowerShell:

```powershell
$env:DIP_API_KEY = "your-key"
```

## License

Code and original documentation in this repository are licensed under the Apache License 2.0. Reference OpenAPI files, public API metadata, and third-party source material may remain subject to their original publishers' terms; see the per-skill reference notes.
