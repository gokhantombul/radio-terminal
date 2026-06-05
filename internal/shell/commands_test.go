package shell

import (
	"bytes"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/ui"
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

func TestCompleterSuggestsLanguageCodes(t *testing.T) {
	cfg := testConfig(t)
	stationSvc := services.NewStationService(cfg)
	if err := stationSvc.Init(); err != nil {
		t.Fatalf("init station service: %v", err)
	}
	audioPlayer := player.NewAudioPlayer(cfg, services.NewNotificationService(services.NewSettingsService(cfg)))
	sh := NewInteractiveShell(stationSvc, audioPlayer)
	completer := newRadioCompleter(sh)

	got, offset := completer.Do([]rune("dil e"), len([]rune("dil e")))
	if offset != 1 {
		t.Fatalf("expected offset 1, got %d", offset)
	}
	if len(got) != 1 || string(got[0]) != "n " {
		t.Fatalf("expected language completion suffix %q, got %q", "n ", got)
	}
}

func TestAllRegisteredCommandsSmoke(t *testing.T) {
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

	commands := RegisterAllCommands(sh, stationSvc, statsSvc, systemSvc, settings, radioBrowser, notifications, audioPlayer)
	commands.webStarted = true
	commands.browserOpener = func(string) {}

	safeArgs := map[string][]string{
		"listele":     {"-n", "3"},
		"turkiye":     {},
		"ulkeler":     {},
		"ulke":        {"-i", "Türkiye"},
		"turler":      {},
		"tur":         {"-i", "Pop"},
		"ara":         {"-s", "Power"},
		"istatistik":  {},
		"durum":       {},
		"sistem":      {},
		"clear":       {},
		"temizle":     {},
		"web":         {},
		"cal":         {},
		"son":         {},
		"dur":         {},
		"durdur":      {},
		"ses":         {},
		"sessiz":      {"ac"},
		"mute":        {"kapat"},
		"sonraki":     {},
		"ileri":       {},
		"onceki":      {},
		"geri":        {},
		"karistir":    {"-u", "ülke-yok"},
		"rastgele":    {"-u", "ülke-yok"},
		"uyku":        {"iptal"},
		"gecmis":      {},
		"kaydet":      {},
		"kayitdur":    {},
		"favori":      {"station-yok"},
		"favoriler":   {},
		"tema":        {},
		"kontrol":     {"station-yok"},
		"ekle":        {"--id", "smoke-test", "--isim", "Smoke Test", "--url", "http://example.invalid/stream"},
		"duzenle":     {"--id", "station-yok"},
		"sil":         {"--id", "station-yok"},
		"iceaktar":    {"-d", filepath.Join(t.TempDir(), "missing.m3u")},
		"bildirim":    {"kapat"},
		"online-ara":  {},
		"online-ekle": {"-n", "1"},
		"dil":         {},
		"lang":        {"-i", "tr"},
	}

	for name := range sh.commands {
		if _, ok := safeArgs[name]; !ok {
			t.Fatalf("missing smoke-test args for registered command %q", name)
		}
	}

	for name, args := range safeArgs {
		cmd, ok := sh.commands[name]
		if !ok {
			t.Fatalf("safe args include unregistered command %q", name)
		}
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.WithOutputAndWidth(&buf, 100, func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("command %q panicked with args %v: %v", name, args, r)
					}
				}()
				cmd.Func(args)
			})
		})
	}

	audioPlayer.Stop()
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
