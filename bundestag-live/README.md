# Bundestag Live

Agent-friendly CLI skill for public Bundestag live/site XML feeds.

## Main command

```text
bundestagctl
```

## Implementations

- Go 2.0: `bin/bundestagctl-2.0.exe`
- Go legacy: `bin/bundestagctl-legacy.exe`
- Python: `python/bundestagctl.py`
- TypeScript / Node.js: `typescript/dist/index.js`

## Common workflows

```text
bundestagctl doctor
bundestagctl members search --name "Amthor" --limit 3
bundestagctl members dossier --id 2022 --grep "Tätigkeiten"
bundestagctl committees search --term "Arbeit" --limit 5
bundestagctl committees dossier --id a11 --member-limit 5 --news-limit 3
bundestagctl plenum conferences --limit 2 --item-limit 5
bundestagctl article get --article-id 1174778 --grep "Meinungsfreiheit"
bundestagctl video feed --content-id 7529016
```

## Important distinction

This is the current Bundestag website/app XML surface. For full parliamentary proceedings, printed papers, plenary protocols, and archival speech research, use `dipctl`.
