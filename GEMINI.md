# Project: Radio Shell (Python)

Terminal-based FM radio player specializing in Turkish and global radio stations. A robust, feature-rich CLI application with interactive shell, recording capabilities, and extensive station management.

## Architectural Overview

The project is structured as a modular, layered CLI application:

1.  **Entry Point (`src/radio/main.py`):** Orchestrates service initialization, dependency injection, and lifecycle management.
2.  **Shell Layer (`src/radio/shell.py`, `commands_*.py`):** Powered by `prompt_toolkit`.
    *   **Interactive REPL:** Custom auto-completion (RadioCompleter) for stations, genres, and countries.
    *   **Command Modules:**
        *   `commands_basic.py`: Listing, search, and navigation.
        *   `commands_playback.py`: Playback controls (volume, play/stop), song history tracking.
        *   `commands_management.py`: CRUD for custom stations, favorites, themes, and online search.
3.  **Service Layer (`src/radio/services/`):** Encapsulated business logic.
    *   `StationService`: Merges built-in JSON stations with user-defined custom stations.
    *   `RadioBrowserService`: Global station discovery via radio-browser.info API.
    *   `SettingsService`: Persists user preferences (volume, last station, notifications).
    *   `StatisticsService`: Tracks listening sessions and usage metrics.
    *   `NotificationService`: Desktop notifications for song changes (Linux/macOS).
    *   `SystemService`: Monitors process memory and child process (ffplay) health.
4.  **Player Layer (`src/radio/player.py`):**
    *   **Playback:** Manages `ffplay` subprocess with asynchronous stderr monitoring for ICY metadata (Song titles).
    *   **Watchdog:** Automated reconnection logic for unstable streams.
    *   **Recording:** Dedicated `ffmpeg` process with real-time transcoding to MP3 (128kbps) and reconnection handling.
5.  **UI Layer (`src/radio/ui.py`):** Rich terminal output management.
    *   Uses `rich` for tables, panels, and styled output.
    *   Supports multiple color themes (Ocean, Forest, Classic, etc.).
    *   Interactive status bar with playback info and recording indicators.

## Tech Stack

*   **Language:** Python 3.10+ (Current: 3.14)
*   **REPL:** `prompt_toolkit` (Interactive shell with history and completion)
*   **UI/Formatting:** `rich` (Tables, colors, progress bars)
*   **Audio Engines:**
    *   `ffplay`: Core playback engine.
    *   `ffmpeg`: Recording and transcoding (MP3).
*   **Testing:** `pytest` (Unit and integration tests)
*   **Containerization:** Docker (Multi-stage build support)

## Development Workflows

### Setup & Run
*   **Launcher:** Execute `./radio.sh` (Linux/macOS) or `radio.bat` (Windows).
*   **Manual Start:** `export PYTHONPATH=. && python3 -m src.radio.main`
*   **Dependencies:** Managed via `requirements.txt`.

### Testing
*   **Run Tests:** `export PYTHONPATH=. && pytest tests/`
*   Ensure 100% coverage for new service logic and command handlers.

## Coding Conventions & Style

*   **Naming:** PEP 8 strict adherence.
*   **UI/UX:** Always use `src/radio/ui.py` wrappers. Never use `print()`.
*   **Concurrency:** Use `threading` for background monitoring; avoid blocking the main REPL loop.
*   **Localization:** Primary language is Turkish. Use UTF-8 for all strings.

## Persistence

All state is stored in `~/.radio-shell/`:
*   `favorites.json`: List of favorite station IDs.
*   `custom-stations.json`: User-defined radio stations.
*   `settings.json`: Volume, last station, notification flags.
*   `stats.json`: Historical listening data.
*   `theme`: Current UI theme name.
*   `recordings/`: High-quality MP3 captures.
