package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/shell"
)

func TestModelRegistersCommands(t *testing.T) {
	m := testModel(t)

	expected := []string{"listele", "cal", "dur", "favori", "online-ara", "web", "dil"}
	for _, name := range expected {
		if _, ok := m.commands[name]; !ok {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
}

func TestFooterRendersStoppedState(t *testing.T) {
	services.L.SetLanguage("tr")
	m := testModel(t)
	m.width = 80

	footer := m.renderFooter()
	if !strings.Contains(footer, "radyo durduruldu") {
		t.Fatalf("expected stopped footer, got %q", footer)
	}
}

func TestNormalizeCommandOutputFitsWidth(t *testing.T) {
	lines := normalizeCommandOutput("\x1b[31m1234567890abcdef\x1b[0m", 8)

	if len(lines) != 1 {
		t.Fatalf("expected one line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "\x1b") {
		t.Fatalf("expected ANSI colors to be preserved, got %q", lines[0])
	}
	if got := ansi.StringWidth(lines[0]); got > 8 {
		t.Fatalf("expected line visual width to fit target, got %d in %q", got, lines[0])
	}
}

func TestDebugTable(t *testing.T) {
	color.NoColor = false
	var buf strings.Builder
	tbl := table.New("NO", "ID", "STATION NAME", "COUNTRY", "GENRE", "FAV").WithWriter(&buf)
	
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		// Replace %s with %s manually if upper casing, or just upper case the formatted string
		formatted := fmt.Sprintf(format, vals...)
		return red.Sprint(strings.ToUpper(formatted))
	})
	
	tbl.AddRow(
		yellow.Sprint(31),
		green.Sprint("tr-paldoga"),
		yellow.Sprint("Pal Doğa"),
		green.Sprint("Türkiye"),
		yellow.Sprint("Türk Halk Müziği"),
		red.Sprint("★"),
	)
	tbl.Print()

	output := buf.String()
	t.Logf("Raw table output:\n%s", output)

	normalized := normalizeCommandOutput(output, 40)
	t.Logf("Normalized output (width 40):\n%s", strings.Join(normalized, "\n"))
}

func TestDebugStationList(t *testing.T) {
	cfg := config.NewRadioConfig()
	stationSvc := services.NewStationService(cfg)
	if err := stationSvc.Init(); err != nil {
		t.Fatalf("init station service: %v", err)
	}
	stations := stationSvc.GetAllStations()
	t.Logf("Total stations loaded: %d", len(stations))
	limit := 40
	if len(stations) < limit {
		limit = len(stations)
	}
	for i := 0; i < limit; i++ {
		t.Logf("Index %d (Number %d): ID=%s, Name=%s, Country=%s, Genre=%s", i, i+1, stations[i].ID, stations[i].Name, stations[i].Country, stations[i].Genre)
	}
}

func testModel(t *testing.T) *Model {
	t.Helper()

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

	m := New(stationSvc, statsSvc, systemSvc, settings, radioBrowser, notifications, audioPlayer)
	shell.RegisterAllCommands(m, stationSvc, statsSvc, systemSvc, settings, radioBrowser, notifications, audioPlayer)
	return m
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
