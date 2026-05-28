# Skill CLI Rename Plan

Goal: make each skill's CLI name match its folder name across docs, binaries, Go, Python, and TypeScript.

1. For each skill folder, replace the old CLI token with the folder name in `SKILL.md`, `README.md`, CLI help text, next actions, package metadata, and generated TypeScript output where present.
2. Rename version-suffixed Windows binaries to `<skill>.exe` and rename each Python entrypoint to `<skill>.py`.
3. Move the former versioned Go source contents into `go/`, then delete the versioned subfolder and `go/README.md`.
4. Delete each `python/README.md`.
5. Run targeted checks for stale CLI names, folder-name skill metadata, moved Go modules, Python compilation, and TypeScript builds where feasible.

## Follow-Up Cleanup Plan

1. Keep each Windows binary named `<skill>.exe`, and update all command examples that point at binaries.
2. Ignore and remove every `<skill>/tests/` folder so the tracked test-result artifacts disappear on the next push.
3. Review each `<skill>/references/*.md` file for real person, party, company, organization, or overly specific example names; replace them with neutral placeholders or general wording.
4. Remove confusing public mentions of old CLI version labels from Markdown and user-facing/code-comment text where they describe the CLI layout rather than real external API names.
5. Recheck TypeScript package/bin metadata and rebuilt/generated output for any missed folder-name CLI normalization.

## Second-Pass Neutrality Audit

1. Re-scan every tracked text file for private contact data, real person names, party names, specific organizations, vendor names, and politically loaded wording.
2. Replace real-world examples with neutral placeholders in repo-owned docs/source while preserving official endpoint names, legal attribution, and source URLs when they are required for citation.
3. Treat API-owner-published OpenAPI files as immutable reference artifacts; scan findings in those files are documented exceptions, not cleanup targets.
4. Rebuild generated TypeScript and Go binaries after source edits.
5. Run a final scanner pass excluding binaries and `.git` metadata.
