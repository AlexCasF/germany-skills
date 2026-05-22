# Tagesschau Skill

Tagesschau current-news context skill with feed search and bounded article expansion.

Primary runtime:

```powershell
& .\skills\tagesschau\bin\tagesschauctl-2.0.exe doctor
```

Alternative runtimes:

```powershell
python skills\tagesschau\python\tagesschauctl.py doctor
node skills\tagesschau\typescript\dist\index.js doctor
```

Start with `SKILL.md`, then use `references/notes.md` and `tests/test-plan.md` for implementation details.
