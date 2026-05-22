# Bundestag Live

Agent-friendly CLI skill for public Bundestag live/site XML feeds.

## Main command

```text
bundestag-live
```

## Implementations

- Go 2.0: `bin/bundestag-live-2.0.exe`
- Python: `python/bundestag-live.py`
- TypeScript / Node.js: `typescript/dist/index.js`

## Common workflows

```text
bundestag-live doctor
bundestag-live members search --name "Amthor" --limit 3
bundestag-live members dossier --id 2022 --grep "Tätigkeiten"
bundestag-live committees search --term "Arbeit" --limit 5
bundestag-live committees dossier --id a11 --member-limit 5 --news-limit 3
bundestag-live plenum conferences --limit 2 --item-limit 5
bundestag-live article get --article-id 1174778 --grep "Meinungsfreiheit"
bundestag-live video feed --content-id 7529016
```

## Important distinction

This is the current Bundestag website/app XML surface. For full parliamentary proceedings, printed papers, plenary protocols, and archival speech research, use `dip-bundestag`.
