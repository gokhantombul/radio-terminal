# ♬ Radio Shell (Go)

**Listen to FM radio stations from Turkey and around the world, directly from your terminal.**

```
  ╔══════════════════════════════════════════════════════════════╗
  ║   ♬  ░░░ RADIO SHELL ░░░  ♬                                  ║
  ║   Terminal FM Radio Player - Türkiye & Dünya                 ║
  ║   Go 1.25 · readline · fatih/color                           ║
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

---

## Command Reference

### Listing Stations

| Command | Description |
|---------|-------------|
| `listele` | List radio stations |
| `turkiye` | List Turkish stations only |
| `ulkeler` | Show available countries |
| `ulke <country>` | Stations from a specific country |
| `turler` | List available music genres |
| `tur <genre>` | Stations of a specific genre |
| `ara <query>` | Search by name, country, or genre |

### Playback

| Command | Description |
|---------|-------------|
| `cal <id\|index>` | Play a station |
| `dur` | Stop playback |
| `durum` | Show currently playing station |
| `ses <0-100>` | Set volume level |
| `sessiz` | Toggle mute/unmute |

### Management

| Command | Description |
|---------|-------------|
| `favori [id]` | Toggle favorite |
| `favoriler` | Show favorite stations |
| `tema <theme>` | Change color theme |
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
