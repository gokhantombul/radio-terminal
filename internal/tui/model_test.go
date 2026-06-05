package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func TestTabCompletesCommandName(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("ca")
	m.refreshSuggestions()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)

	if got := m.input.Value(); got != "cal " {
		t.Fatalf("expected tab to complete command, got %q", got)
	}
}

func TestTabCompletesCommandArgument(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("cal tr")
	m.refreshSuggestions()

	expected := firstCompletion(t, m.completionSuggestions("cal tr"))
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)

	if got := m.input.Value(); got != expected {
		t.Fatalf("expected tab to complete station argument to %q, got %q", expected, got)
	}
}

func TestTabCompletesStationByIDSubstring(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("cal power")
	m.refreshSuggestions()
	if got := m.input.MatchedSuggestions(); len(got) != 0 {
		t.Fatalf("expected textinput prefix suggestions to be empty for substring match, got %v", got)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)

	if got := m.input.Value(); got != "cal tr-powerturk " {
		t.Fatalf("expected substring tab completion to prefer powerturk, got %q", got)
	}
}

func TestRepeatedTabCyclesStationSubstringMatches(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("cal power")
	m.refreshSuggestions()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)
	if got := m.input.Value(); got != "cal tr-powerturk " {
		t.Fatalf("expected first tab to complete powerturk, got %q", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)
	if got := m.input.Value(); got != "cal tr-powerfm " {
		t.Fatalf("expected second tab to cycle to powerfm, got %q", got)
	}
}

func TestTabCompletesStationByNormalizedName(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("cal doga")
	m.refreshSuggestions()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)

	if got := m.input.Value(); got != "cal tr-paldoga " {
		t.Fatalf("expected normalized name completion, got %q", got)
	}
}

func TestTabCompletesFlag(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("online-ara -")
	m.refreshSuggestions()

	expected := firstCompletion(t, m.completionSuggestions("online-ara -"))
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*Model)

	if got := m.input.Value(); got != expected {
		t.Fatalf("expected tab to complete flag to %q, got %q", expected, got)
	}
}

func TestDownCyclesInputSuggestions(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.input.SetValue("cal tr")
	m.refreshSuggestions()
	suggestions := m.input.MatchedSuggestions()
	if len(suggestions) < 2 {
		t.Fatalf("expected multiple suggestions, got %v", suggestions)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*Model)

	if got := m.input.Value(); got != "cal tr" {
		t.Fatalf("expected down to keep input text, got %q", got)
	}
	if got := m.input.CurrentSuggestion(); got != suggestions[1] {
		t.Fatalf("expected down to select %q, got %q", suggestions[1], got)
	}
}

func TestUpRecallsHistoryWhenNoSuggestionsMatch(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.commandHist = []string{"listele", "help"}
	m.input.SetValue("zz")
	m.refreshSuggestions()
	if got := m.input.MatchedSuggestions(); len(got) != 0 {
		t.Fatalf("expected no matched suggestions, got %v", got)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(*Model)

	if got := m.input.Value(); got != "help" {
		t.Fatalf("expected up to recall history when no suggestions match, got %q", got)
	}
}

func TestCtrlNCyclesSuggestionsWhileTyping(t *testing.T) {
	m := testModel(t)
	m.input.Focus()
	m.selected = 0
	m.input.SetValue("cal tr")
	m.refreshSuggestions()
	suggestions := m.input.MatchedSuggestions()
	if len(suggestions) < 2 {
		t.Fatalf("expected multiple suggestions, got %v", suggestions)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = updated.(*Model)

	if got := m.selected; got != 0 {
		t.Fatalf("expected ctrl+n not to move station selection while typing, got selected index %d", got)
	}
	if got := m.input.CurrentSuggestion(); got != suggestions[1] {
		t.Fatalf("expected ctrl+n to select %q, got %q", suggestions[1], got)
	}
}

func TestViewFitsTerminalSizes(t *testing.T) {
	services.L.SetLanguage("tr")
	m := testModel(t)
	m.commandLog = []string{
		infoStyle.Render("❯ listele"),
		"  Bu satır özellikle uzun tutuldu; dar terminallerde taşmadan kırpılmalı ve paneller üst üste binmemeli.",
	}
	m.refreshOutput()
	m.busy = true
	m.busyCommand = "online-ara cok uzun arama metni"
	m.input.SetValue("li")

	cases := []struct {
		name   string
		width  int
		height int
	}{
		{name: "wide", width: 120, height: 32},
		{name: "standard", width: 80, height: 24},
		{name: "narrow", width: 64, height: 24},
		{name: "compact", width: 50, height: 18},
		{name: "short", width: 40, height: 12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m.width = tc.width
			m.height = tc.height
			m.resize()

			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > tc.height {
				t.Fatalf("view height = %d, want <= %d\n%s", len(lines), tc.height, view)
			}
			for i, line := range lines {
				if got := ansi.StringWidth(line); got > tc.width {
					t.Fatalf("line %d width = %d, want <= %d\n%q\n%s", i+1, got, tc.width, line, view)
				}
			}
		})
	}
}

func TestLayoutUsesWiderStationPanel(t *testing.T) {
	m := testModel(t)
	m.width = 80
	m.height = 24

	layout := m.calculateLayout(1, 3, 1)

	if !layout.sideBySide {
		t.Fatalf("expected side-by-side layout")
	}
	if layout.leftWidth < 33 {
		t.Fatalf("expected wider left station panel, got width %d", layout.leftWidth)
	}
}

func TestPlaySelectedShowsLoadingState(t *testing.T) {
	m := testModel(t)
	m.width = 100
	m.height = 24
	m.resize()
	if len(m.stations) == 0 {
		t.Fatal("expected stations")
	}

	selected := m.stations[m.selected]
	_ = m.playSelectedCommand()

	id, name, loading := m.stationLoading()
	if !loading {
		t.Fatal("expected loading state after selecting a station")
	}
	if id != selected.ID || name != selected.Name {
		t.Fatalf("loading state = (%q, %q), want (%q, %q)", id, name, selected.ID, selected.Name)
	}
	if got := m.renderInput(); !strings.Contains(got, "Yükleniyor") {
		t.Fatalf("expected input to show loading, got %q", got)
	}
	if got := m.renderFooter(); !strings.Contains(got, "Yükleniyor") {
		t.Fatalf("expected footer to show loading, got %q", got)
	}
	if got := m.renderStationPanel(50, 12); !strings.Contains(got, "Yükleniyor") {
		t.Fatalf("expected station panel to show loading, got %q", got)
	}
	if got := m.renderCommandOutputBox(60, 12); !strings.Contains(got, "Bağlanıyor") || !strings.Contains(got, "[") {
		t.Fatalf("expected command output to show loading bar, got %q", got)
	}

	m.clearExpiredLoading(time.Now().Add(3 * time.Second))
	if _, _, loading := m.stationLoading(); loading {
		t.Fatal("expected expired loading state to clear")
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

func firstCompletion(t *testing.T, suggestions []string) string {
	t.Helper()
	if len(suggestions) == 0 {
		t.Fatal("expected at least one completion suggestion")
	}
	return uniqueStrings(suggestions)[0]
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
