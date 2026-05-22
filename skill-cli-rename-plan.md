# Skill CLI Rename Plan

Goal: make each skill's CLI name match its folder name across docs, binaries, Go, Python, and TypeScript.

1. For each skill folder, replace the old CLI token with the folder name in `SKILL.md`, `README.md`, CLI help text, next actions, package metadata, and generated TypeScript output where present.
2. Rename `bin/*-2.0.exe` to `<skill>-2.0.exe` and rename each Python entrypoint to `<skill>.py`.
3. Move the former versioned Go source contents into `go/`, then delete the versioned subfolder and `go/README.md`.
4. Delete each `python/README.md`.
5. Run targeted checks for stale CLI names, folder-name skill metadata, moved Go modules, Python compilation, and TypeScript builds where feasible.
