package config

import (
	"os"
	"path/filepath"
)

type PlayerConfig struct {
	Command string
	Args    []string
}

type StationsConfig struct {
	File string
}

type RadioConfig struct {
	Player             PlayerConfig
	Stations           StationsConfig
	AppDir             string
	FavoritesFile      string
	CustomStationsFile string
	SettingsFile       string
	RecordingsDir      string
	StatsFile          string
	WebPIDFile         string
}

func NewRadioConfig() *RadioConfig {
	homeDir, _ := os.UserHomeDir()
	appDir := filepath.Join(homeDir, ".radio-shell")

	return &RadioConfig{
		Player: PlayerConfig{
			Command: "ffplay",
			Args:    []string{"-nodisp", "-hide_banner", "-loglevel", "quiet", "-autoexit"},
		},
		Stations: StationsConfig{
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

func (c *RadioConfig) EnsureDirs() error {
	if err := os.MkdirAll(c.AppDir, 0755); err != nil {
		return err
	}
	return os.MkdirAll(c.RecordingsDir, 0755)
}
