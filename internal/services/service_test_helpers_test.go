package services

import (
	"path/filepath"
	"radio-shell/internal/config"
	"testing"
)

func testConfig(t *testing.T) *config.RadioConfig {
	t.Helper()

	appDir := t.TempDir()
	return &config.RadioConfig{
		Player: config.PlayerConfig{
			Command: "ffplay",
			Args:    []string{"-nodisp", "-hide_banner", "-loglevel", "quiet", "-autoexit"},
		},
		Stations: config.StationsConfig{
			File: "stations.json",
		},
		AppDir:             appDir,
		FavoritesFile:      filepath.Join(appDir, "favorites.json"),
		CustomStationsFile: filepath.Join(appDir, "custom-stations.json"),
		SettingsFile:       filepath.Join(appDir, "settings.json"),
		RecordingsDir:      filepath.Join(appDir, "recordings"),
		StatsFile:          filepath.Join(appDir, "stats.json"),
		WebPIDFile:         filepath.Join(appDir, "web.pid"),
	}
}
