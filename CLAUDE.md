# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Run, test, develop

- Run the app: `./radio.sh` — creates `venv/` on first run, installs `requirements.txt`, sets `PYTHONPATH`, and execs `python3 -m src.radio.main`.
- Run without the launcher: `PYTHONPATH=. ./venv/bin/python3 -m src.radio.main` (the shell needs an interactive TTY).
- Tests: `pytest tests/` from the repo root. Single test: `pytest tests/test_completer.py::test_command_completion_substring`.
- Adding a dependency: append to `requirements.txt`; the launcher only `pip install`s on first venv creation, so existing checkouts need a manual `./venv/bin/pip install -r requirements.txt` after pulling.
- External runtime dependency: `ffplay` (audio playback) and `ffmpeg` (used by the `kaydet` recording command). `radio.sh` aborts if `ffplay` is missing.

## Heads-up: README.md is stale

`README.md` still describes the previous Java/Spring Shell/Maven implementation (`pom.xml`, `mvn clean package`, `src/main/java/...`). The current source of truth is the Python code under `src/radio/`. `GEMINI.md` describes the Python layout accurately. Don't edit Java paths in README without checking that they map to anything that exists.

## Architecture

Everything is wired manually in `src/radio/main.py` — no DI framework. The flow is: build `RadioConfig`, instantiate services, instantiate `AudioPlayer`, instantiate `InteractiveShell`, then construct each `*Commands` class which registers its handlers against the shell as a side effect. `shell.run(player)` blocks; `player.stop()` runs on exit.

Four layers, with strict direction of dependency (commands → services/player → models/config):

1. **Shell (`shell.py`)** — `prompt_toolkit` REPL. `InteractiveShell.register(name, func, desc, category)` is the extension point; the `*Commands` constructors call it for each command. `RadioCompleter` does context-aware completion: for `cal`/`kontrol`/`favori`/`sil` it completes station IDs/names, for `ulke`/`tur`/`tema` it completes the respective vocabulary. The bottom toolbar polls `AudioPlayer` state every second (`refresh_interval=1.0`).
2. **Commands (`commands_basic.py`, `commands_playback.py`, `commands_management.py`)** — each module is a class that takes its dependencies in `__init__` and registers Turkish-named commands. Arguments are parsed with `argparse` (catch `SystemExit` and show a usage hint via `ui.print_error`). `PlaybackCommands` hooks `player.on_song_change` to maintain a 50-entry song history and `BasicCommands.last_list` is shared so `sonraki`/`onceki` (next/prev) can iterate over whatever list was last shown.
3. **Services (`src/radio/services/`)** — pure data/IO logic, no UI imports.
   - `StationService` merges built-in stations from `src/main/resources/stations.json` (legacy Java path, still used by Python via fallback search in `_load_internal_stations`) with user custom stations from `~/.radio-shell/custom-stations.json`. The favorite flag is computed at read time by intersecting with `favorites.json`.
   - `SettingsService` persists `UserSettings` (volume, last station, notifications) to `~/.radio-shell/settings.json`.
   - `StatisticsService` records sessions ≥30s to `~/.radio-shell/stats.json`.
   - `RadioBrowserService` queries `de1.api.radio-browser.info` for the `online-ara`/`online-ekle` commands.
   - `NotificationService` shells out to `osascript` (macOS) or `notify-send` (Linux).
   - `SystemService` uses `psutil` to report process + child (ffplay) memory.
4. **Player (`player.py`)** — `AudioPlayer` spawns `ffplay` as a subprocess with `-loglevel info` so it can scrape ICY metadata (`StreamTitle:`) and stream info (codec, sample rate, bitrate) from stderr in a background thread. A second watchdog thread restarts ffplay up to 3 times on unexpected exit. Recording is a separate `ffmpeg` subprocess that transcodes the stream to MP3 (128k) and writes to `~/.radio-shell/recordings/`.
5. **UI (`ui.py`)** — single `rich.Console`, four named themes, persisted to `~/.radio-shell/theme`. All user-facing output should go through `ui.print_error/success/info` or the table/banner helpers — do not use bare `print()`. Strings are Turkish; keep UTF-8 (Turkish characters like `İ`, `ü`, `ö` appear in headers).

## Persistent state

All under `~/.radio-shell/`: `favorites.json`, `custom-stations.json`, `settings.json`, `stats.json`, `theme`, `recordings/`. `RadioConfig.ensure_dirs()` only creates `recordings/`; the JSON files are created lazily on first save by each service.

## Conventions

- Command names and user-facing strings are Turkish. Internal identifiers, comments, and docstrings are English.
- New command? Add it to a `commands_*.py` module and call `shell.register(...)` in that class's constructor. Also add it to the category lists in `InteractiveShell.print_help` (`shell.py`) or it won't show up in `help`.
- Catch `SystemExit` around `argparse.parse_args(args)` — `argparse` calls `sys.exit` on bad input, which would otherwise kill the REPL.
