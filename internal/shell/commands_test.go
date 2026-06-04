package shell

import (
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"testing"
)

func TestRegisterAllCommandsIncludesPythonParityCommands(t *testing.T) {
	cfg := testConfig(t)
	settings := services.NewSettingsService(cfg)
	stationSvc := services.NewStationService(cfg)
	if err := stationSvc.Init(); err != nil {
		t.Fatalf("init station service: %v", err)
	}
	statsSvc := services.NewStatisticsService(cfg)
	systemSvc := services.NewSystemService()
	radioBrowser := services.NewRadioBrowserService()
	notifications := services.NewNotificationService(settings)
	audioPlayer := player.NewAudioPlayer(cfg, notifications)
	sh := NewInteractiveShell(stationSvc, audioPlayer)

	RegisterAllCommands(sh, stationSvc, statsSvc, systemSvc, settings, radioBrowser, notifications, audioPlayer)

	expected := []string{
		"listele", "turkiye", "ulkeler", "ulke", "turler", "tur", "ara", "web",
		"cal", "son", "dur", "durdur", "ses", "sessiz", "mute", "sonraki", "ileri",
		"onceki", "geri", "karistir", "rastgele", "uyku", "gecmis",
		"kaydet", "kayitdur",
		"favori", "favoriler", "tema", "kontrol", "ekle", "duzenle", "sil",
		"iceaktar", "bildirim", "online-ara", "online-ekle", "dil", "lang",
	}
	for _, name := range expected {
		if _, ok := sh.commands[name]; !ok {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
}

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
