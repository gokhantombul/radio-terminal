# AGENTS.md

This file provides guidance to Codex when working with this repository.

## Run, Test, Develop

- Build: `go build -o radio ./cmd/radio`
- Run existing binary: `./radio`
- Run without building: `go run ./cmd/radio`
- Start web UI: `./radio --web`
- Start web UI in foreground: `./radio --web --foreground`
- Stop background web UI: `./radio --kill`
- Tests: `go test ./...`
- If Go tries to write cache files outside the workspace, use
  `GOCACHE=/tmp/go-build go test ./...`
- Runtime dependencies: `ffplay` for playback and `ffmpeg` for MP3 recording.
- Add dependencies with `go get` or `go mod tidy`; commit `go.mod` and
  `go.sum` together.

Do not use old Python instructions in this repository. There is no active
`src/radio`, `venv`, `requirements.txt`, `pytest`, `radio.sh`, Maven, or Spring
Shell implementation in the current tree.

## Architecture

The app is wired manually. `cmd/radio/main.go` calls `internal/app.Run`, which
parses flags, builds `RadioConfig`, creates `~/.radio-shell` paths, prompts for
language on first interactive run, constructs services, creates `AudioPlayer`,
and starts either the Bubble Tea TUI or the Gin web server.

Layers:

1. **App (`internal/app`)** - top-level lifecycle, flags, language bootstrap,
   background web process handling, and browser opening.
2. **TUI (`internal/tui`)** - default full-screen Bubble Tea interface. It
   implements `shell.CommandHost`, captures `internal/ui` output into the right
   panel, owns command history, suggestions, station selection, and footer
   refresh.
3. **Shell (`internal/shell`)** - central command registry and handlers.
   `RegisterAllCommands` is the source of truth for command names, aliases,
   categories, descriptions, session hooks, song history, and sleep timer.
4. **Services (`internal/services`)** - data and OS integrations: stations,
   settings, statistics, RadioBrowser, notifications, system metrics, and
   localization.
5. **Player (`internal/player`)** - `ffplay` subprocess management, ICY metadata
   parsing, stream info parsing, watchdog restarts, and `ffmpeg` recording.
6. **Web (`internal/web`)** - Gin router and embedded static assets under
   `internal/web/static`.
7. **UI (`internal/ui`)** - terminal output helpers and themes.

Keep dependencies directed downward: app wires dependencies, TUI/shell command
layers call services/player, and services depend only on config/models plus
external infrastructure packages.

## Command Surface

Registered command names:

- Listing/search: `listele`, `turkiye`, `ulkeler`, `ulke`, `turler`, `tur`,
  `ara`, `online-ara`
- Playback: `cal`, `son`, `dur`, `durdur`, `durum`, `ses`, `sessiz`, `mute`,
  `sonraki`, `ileri`, `onceki`, `geri`, `karistir`, `rastgele`, `uyku`,
  `gecmis`
- Recording: `kaydet`, `kayitdur`
- Management: `favori`, `favoriler`, `tema`, `kontrol`, `ekle`, `duzenle`,
  `sil`, `iceaktar`, `bildirim`, `online-ekle`, `dil`, `lang`, `istatistik`,
  `sistem`, `web`, `temizle`, `clear`
- General shell commands: `help`, `?`, `exit`, `q`, `quit`

Argument parsing uses `flag.FlagSet` with `flag.ContinueOnError`; flag output is
discarded, so handlers should print concise usage errors through `internal/ui`.

## Persistent State

All persistent user data is under `~/.radio-shell/`:

- `favorites.json`
- `custom-stations.json`
- `settings.json`
- `stats.json`
- `theme`
- `recordings/`
- `web.pid`

`RadioConfig.EnsureDirs` creates the app directory and recordings directory.
JSON files are created lazily when services save data. Built-in stations are
embedded from `internal/services/stations.json`.

## Conventions

- User-facing commands and default UI text are Turkish; preserve UTF-8.
- New command: add a handler in `internal/shell/commands.go`, register it in
  `RegisterAllCommands`, add localization strings in
  `internal/services/localization.go`, and update completions if needed.
- Completion logic exists in `shell.FlagSuggestions`, `radioCompleter`, and the
  TUI completion helpers.
- Command output should use `internal/ui` helpers, not direct `fmt.Print`, so
  TUI command capture keeps working.
- Services should not import TUI or terminal UI packages.
- Statistics record sessions only when playback duration is at least 30 seconds.
- Web system info keeps the JSON key `python_version` for compatibility while
  returning the Go runtime version.
- `scripts/install-command.sh` and `scripts/install-command.ps1` still reference
  legacy launchers. Fix them before relying on them.
