# ♬ Radio Shell (Go)

**Listen to FM radio stations from Turkey and around the world, directly from your terminal.**

```
  ╔══════════════════════════════════════════════════════════════╗
  ║   ♬  ░░░ RADIO SHELL ░░░  ♬                                  ║
  ║   Terminal FM Radio Player - Türkiye & Dünya                 ║
  ║   Go 1.25 · Bubble Tea · Lip Gloss                            ║
  ╚══════════════════════════════════════════════════════════════╝
```

---

## Requirements

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | [go.dev](https://go.dev) |
| ffmpeg (ffplay) | Any | See below |

---

## Setup & Running

### Build

```bash
go build -o radio ./cmd/radio
```

### Run

```bash
./radio
```

The default terminal experience is a full-screen Bubble Tea TUI with a station
list, command output panel, command input with suggestions, and a live footer
showing station, song metadata, country, genre, stream info, volume, elapsed
time, and recording state. On first interactive run, the app asks for a UI
language and persists it in `~/.radio-shell/settings.json`.

### TUI Keys

| Key | Action |
|-----|--------|
| `Enter` | Run the typed command; with an empty input, play the selected station |
| `↑` / `↓` | Move station selection when input is empty; browse command history while typing |
| `Ctrl+N` / `Ctrl+P` | Move station selection |
| `Ctrl+F` | Toggle favorite for selected station |
| `Ctrl+S` | Stop playback |
| `Ctrl+L` | Clear command output |
| `PgUp` / `PgDown` | Scroll command output |
| `Ctrl+C` / `Esc` | Quit |

### Web Interface

```bash
./radio --web              # start in background and open browser
./radio --web --foreground # run the web server in the current terminal
./radio --kill             # stop the background web server
```

---

## Project Layout

```text
cmd/radio/          Thin binary entrypoint
internal/app/       Application wiring, flags, web lifecycle
internal/tui/       Bubble Tea terminal UI
internal/shell/     Command registry and command handlers
internal/player/    ffplay/ffmpeg playback and recording process control
internal/services/  Stations, settings, stats, RadioBrowser, notifications
internal/web/       Gin web server and embedded static UI
internal/ui/        Shared terminal output helpers and themes
```

---

## Command Reference

### Listing Stations

| Command | Description |
|---------|-------------|
| `listele` | List radio stations |
| `turkiye` | List Turkish stations only |
| `ulkeler` | Show available countries |
| `ulke -i <country>` | Stations from a specific country |
| `turler` | List available music genres |
| `tur -i <genre>` | Stations of a specific genre |
| `ara -s <query>` | Search by name, country, or genre |
| `online-ara -s <query>` | Search RadioBrowser online directory |

### Playback

| Command | Description |
|---------|-------------|
| `cal <id\|index>` | Play a station |
| `son` | Play the last played station |
| `dur` | Stop playback |
| `durum` | Show currently playing station |
| `ses -s <0-100>` | Set volume level |
| `sessiz [ac\|kapat]` | Toggle mute/unmute |
| `sonraki` / `ileri` | Play next station from the last list |
| `onceki` / `geri` | Play previous station from the last list |
| `karistir [-u country] [-t genre]` | Play a random station |
| `uyku -d <minutes>` / `uyku iptal` | Start or cancel sleep timer |
| `gecmis` | Show recent song metadata |

### Management

| Command | Description |
|---------|-------------|
| `favori [id]` | Toggle favorite |
| `favoriler` | Show favorite stations |
| `tema <theme>` | Change color theme |
| `kontrol [id]` | Check stream URL availability |
| `ekle --id <id> --isim <name> --url <url>` | Add a custom station |
| `duzenle --id <id> [--isim ...] [--url ...]` | Edit a custom station |
| `sil --id <id>` | Delete a custom station |
| `iceaktar -d <playlist.m3u>` | Import stations from a playlist |
| `bildirim [ac\|kapat]` | Toggle desktop notifications |
| `online-ekle -n <no>` | Add a station from online search results |
| `dil -i <code>` / `lang -i <code>` | Change application language |
| `web` | Start the web interface on `127.0.0.1:8765` |
| `sistem` | Show system information |
| `temizle` | Clear the terminal screen |

### Other

| Command | Description |
|---------|-------------|
| `help` / `?` | Show help menu |
| `exit` / `q` | Quit the application |

---

## License

MIT License.
