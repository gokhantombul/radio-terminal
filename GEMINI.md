# Project: Radio Terminal (Go)

Terminal FM radio player for local and global stations. The current codebase is
Go, with a Bubble Tea TUI as the default interface and an optional Gin web UI.

## Run, Test, Develop

- Build: `go build -o radio ./cmd/radio`
- Run existing binary: `./radio`
- Run without building: `go run ./cmd/radio`
- Start web UI: `./radio --web`
- Start web UI in foreground: `./radio --web --foreground`
- Stop background web UI: `./radio --kill`
- Tests: `go test ./...`
- If the Go cache is not writable: `GOCACHE=/tmp/go-build go test ./...`
- Runtime dependencies: `ffplay` for playback and `ffmpeg` for the `kaydet`
  recording command.
- Adding dependencies: use `go get` or `go mod tidy`; commit `go.mod` and
  `go.sum` together.

There is no Python `src/radio`, `venv`, `requirements.txt`, or `pytest` test
suite in this checkout. Do not document or edit old Python paths unless they are
reintroduced deliberately.

## Architecture

Everything is wired manually from `cmd/radio/main.go` through `internal/app`.
`app.Run` parses flags, builds `RadioConfig`, ensures `~/.radio-shell`, selects
language on first interactive run, instantiates services, creates `AudioPlayer`,
then starts either the Bubble Tea TUI or the Gin web server.

Main layers:

1. **App (`internal/app`)** - CLI flags, language bootstrap, web background
   process management, browser opening, and top-level lifecycle.
2. **TUI (`internal/tui`)** - full-screen Bubble Tea model. It implements
   `shell.CommandHost`, captures command output with `ui.WithOutputAndWidth`,
   maintains the station panel, command output panel, suggestions, history, and
   footer status.
3. **Shell commands (`internal/shell`)** - central command registry and command
   handlers. `RegisterAllCommands` is the source of truth for command names,
   categories, descriptions, aliases, song-history tracking, sleep timer, and
   session recording hooks.
4. **Services (`internal/services`)** - data and OS integration with no TUI
   imports. Includes station storage, settings, statistics, notifications,
   system stats, localization, and RadioBrowser API search.
5. **Player (`internal/player`)** - manages `ffplay` playback, scrapes ICY
   metadata and stream info from stderr, restarts failed playback up to three
   times, and records active streams with `ffmpeg`.
6. **Web (`internal/web`)** - Gin API plus embedded static files from
   `internal/web/static`. Uses the same player, settings, station, and system
   services as the TUI.
7. **UI (`internal/ui`)** - shared terminal output helpers and terminal themes.

Dependency direction should stay one-way: app wires everything, command/UI
layers call services/player, and services depend only on config/models and
standard or external infrastructure packages.

## Commands

`RegisterAllCommands` currently registers these user-facing commands:

- Listing/search: `listele`, `turkiye`, `ulkeler`, `ulke`, `turler`, `tur`,
  `ara`, `online-ara`
- Playback: `cal`, `son`, `dur`, `durdur`, `durum`, `ses`, `sessiz`, `mute`,
  `sonraki`, `ileri`, `onceki`, `geri`, `karistir`, `rastgele`, `uyku`,
  `gecmis`
- Recording: `kaydet`, `kayitdur`
- Management: `favori`, `favoriler`, `tema`, `kontrol`, `ekle`, `duzenle`,
  `sil`, `iceaktar`, `bildirim`, `online-ekle`, `dil`, `lang`, `istatistik`,
  `sistem`, `web`, `temizle`, `clear`
- General commands handled by shells: `help`, `?`, `exit`, `q`, `quit`

Command arguments are parsed with Go `flag.FlagSet` using `flag.ContinueOnError`
and discarded flag output. Keep usage errors explicit through `internal/ui`.

## Persistent State

All user state lives under `~/.radio-shell/`:

- `favorites.json` - favorite station IDs.
- `custom-stations.json` - custom station list in `models.StationList` format.
- `settings.json` - volume, muted flag, last station, notifications, language.
- `stats.json` - listening sessions of at least 30 seconds.
- `theme` - selected terminal theme.
- `recordings/` - MP3 recordings created by `kaydet`.
- `web.pid` - PID and URL for the background web server.

Built-in stations are embedded from `internal/services/stations.json`.

## Coding Conventions

- User-facing command names and default text are Turkish. Keep UTF-8 Turkish
  characters intact.
- Add new commands in `internal/shell/commands.go` by registering them in
  `RegisterAllCommands`.
- Add or update command description keys and translated strings in
  `internal/services/localization.go`.
- Update completion logic in `shell.FlagSuggestions`, `radioCompleter`, and TUI
  completion helpers when new flags or argument vocabularies are added.
- Use `internal/ui` output helpers for command output. The TUI relies on this
  to capture command output into the right panel.
- Keep services free of UI imports.
- Do not rename web JSON keys casually; `SystemService.GetWebInfo` still emits
  `python_version` for web compatibility even though the value is the Go
  runtime version.
- Existing `scripts/install-command.sh` and `scripts/install-command.ps1`
  reference old launchers. Update the scripts before documenting or using them.
