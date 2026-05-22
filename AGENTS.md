# AGENTS.md

## Purpose

This repository is a toolbox for agents that research German public data. It contains self-contained skill folders with:

- `SKILL.md` instructions for progressive tool discovery
- OpenAPI or documentation references
- Go, Python, and TypeScript/Node.js CLI implementations
- JSON-first commands for source-rich research

Treat every API family as read-only. Do not design or run flows that perform external side effects.

## Agent Usage Rule

Do not ingest this whole repository into context.

Use progressive disclosure:

1. Identify the likely API family.
2. Read that folder's `SKILL.md`.
3. Run the CLI's root `--help`.
4. Inspect subcommand help only as needed.
5. Prefer narrow, source-preserving commands over broad dumps.
6. Link or cite sources returned by the CLI.

## Tool Families

| Folder | CLI | Typical first command |
| --- | --- | --- |
| `abgeordnetenwatch` | `abgeordnetenwatch` | `abgeordnetenwatch doctor` |
| `bundeshaushalt` | `bundeshaushalt` | `bundeshaushalt doctor` |
| `bundesrat-live` | `bundesrat-live` | `bundesrat-live doctor` |
| `bundestag-live` | `bundestag-live` | `bundestag-live doctor` |
| `bundestag-lobbyregister` | `bundestag-lobbyregister` | `bundestag-lobbyregister doctor` |
| `dashboard-deutschland` | `dashboard-deutschland` | `dashboard-deutschland doctor` |
| `destatis` | `destatis` | `destatis doctor` |
| `deutschlandatlas` | `deutschlandatlas` | `deutschlandatlas doctor` |
| `dip-bundestag` | `dip-bundestag` | `dip-bundestag doctor` |
| `rechtsinformationen-bund` | `rechtsinformationen-bund` | `rechtsinformationen-bund doctor` |
| `regionalatlas` | `regionalatlas` | `regionalatlas doctor` |
| `tagesschau` | `tagesschau` | `tagesschau doctor` |

## CLI Flavor One-Liners

Use `dip-bundestag` as the example pattern; replace the folder and binary names from the table above.

Go from source:

```bash
git clone https://github.com/AlexCasF/germany-skills.git && cd germany-skills/dip-bundestag/go && go run . doctor
```

Python:

```bash
git clone https://github.com/AlexCasF/germany-skills.git && cd germany-skills && python dip-bundestag/python/dip-bundestag.py doctor
```

TypeScript/Node.js:

```bash
git clone https://github.com/AlexCasF/germany-skills.git && cd germany-skills && npm --prefix dip-bundestag/typescript ci && npm --prefix dip-bundestag/typescript run build && node dip-bundestag/typescript/dist/index.js doctor
```

Windows binary, where available:

```powershell
git clone https://github.com/AlexCasF/germany-skills.git; cd germany-skills; .\dip-bundestag\bin\dip-bundestag.exe doctor
```

## Agent Installation Patterns

### Codex

Best project-local pattern:

```bash
git clone https://github.com/AlexCasF/germany-skills.git .agent/germany-skills
```

Then tell Codex: read `.agent/germany-skills/AGENTS.md`, use `.agent/germany-skills/<skill>/SKILL.md` as needed, and run CLIs from that checkout.

Optional user-level skill linking on Windows PowerShell:

```powershell
git clone https://github.com/AlexCasF/germany-skills.git "$env:USERPROFILE\germany-skills"; New-Item -ItemType Directory -Force "$env:USERPROFILE\.codex\skills"; Get-ChildItem "$env:USERPROFILE\germany-skills" -Directory | Where-Object { Test-Path "$($_.FullName)\SKILL.md" } | ForEach-Object { if (-not (Test-Path "$env:USERPROFILE\.codex\skills\$($_.Name)")) { New-Item -ItemType Junction -Path "$env:USERPROFILE\.codex\skills\$($_.Name)" -Target $_.FullName } }
```

### Claude Code

Project-local pattern:

```bash
git clone https://github.com/AlexCasF/germany-skills.git .agent/germany-skills
```

Then add or tell Claude Code: read `.agent/germany-skills/AGENTS.md`; for a specific task, read only the relevant `.agent/germany-skills/<skill>/SKILL.md`.

If your Claude Code setup supports local skill folders, link each folder containing `SKILL.md` into `.claude/skills`.

### Google Agents CLI / ADK

Use this repository as a bundled skill/tool asset:

```bash
git clone https://github.com/AlexCasF/germany-skills.git skills
```

Bundle `skills/` into the agent runtime image or deployment package, compile the Go CLIs into the runtime `bin/` directory, and expose shell execution through your normal ADK tool boundary.

### Hermes, OpenClaw, Pi, Picoclaw, And Similar Computer Agents

These agents generally work best with a workspace checkout and bash access:

```bash
git clone https://github.com/AlexCasF/germany-skills.git skills/germany
```

Point the agent at `skills/germany/AGENTS.md`. Let it read only the `SKILL.md` for the API family it intends to use. If the agent supports a persistent workspace, keep generated reports and intermediate files outside this repository unless you intentionally want to commit them.

## Output Expectations

Prefer commands that return structured JSON. Final answers should preserve:

- source URLs
- identifiers such as Drucksache numbers, ELI/ECLI values, politician IDs, or dataset codes
- timestamps and data vintages
- uncertainty or incompleteness warnings from the CLI

## Safety And Fair Use

- Use small limits for discovery.
- Avoid broad crawling unless the user explicitly asks and the source permits it.
- Respect per-skill `references/rate-limits-and-terms.md`.
- Do not put API keys into commands if an environment variable is supported.
- Do not echo secrets into final answers or logs.

