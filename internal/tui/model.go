package tui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-shellwords"

	"radio-shell/internal/models"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/shell"
	"radio-shell/internal/ui"
)

type Model struct {
	mu sync.RWMutex

	commands map[string]shell.ShellCommand
	onExit   func()
	exitOnce sync.Once

	stationService  *services.StationService
	statsService    *services.StatisticsService
	systemService   *services.SystemService
	settingsService *services.SettingsService
	radioBrowser    *services.RadioBrowserService
	notificationSvc *services.NotificationService
	player          *player.AudioPlayer

	input  textinput.Model
	output viewport.Model

	width  int
	height int

	stations      []models.RadioStation
	lastList      []models.RadioStation
	selected      int
	scrollStart   int
	commandLog    []string
	commandHist   []string
	historyCursor int
	completionSet []string
	completionIdx int
	busy          bool
	busyCommand   string
	message       string
}

type tickMsg time.Time

type commandResultMsg struct {
	line   string
	output string
	err    error
}

type screenLayout struct {
	bodyHeight           int
	sideBySide           bool
	leftWidth            int
	rightWidth           int
	stationHeight        int
	rightHeight          int
	outputViewportWidth  int
	outputViewportHeight int
}

type palette struct {
	bg       lipgloss.Color
	panel    lipgloss.Color
	panelAlt lipgloss.Color
	border   lipgloss.Color
	text     lipgloss.Color
	muted    lipgloss.Color
	cyan     lipgloss.Color
	amber    lipgloss.Color
	coral    lipgloss.Color
	green    lipgloss.Color
	red      lipgloss.Color
}

var p = palette{
	bg:       lipgloss.Color("#101318"),
	panel:    lipgloss.Color("#171B22"),
	panelAlt: lipgloss.Color("#1E242D"),
	border:   lipgloss.Color("#3B4656"),
	text:     lipgloss.Color("#E6EDF3"),
	muted:    lipgloss.Color("#8B98A8"),
	cyan:     lipgloss.Color("#2FD1C5"),
	amber:    lipgloss.Color("#F6C85F"),
	coral:    lipgloss.Color("#FF6B6B"),
	green:    lipgloss.Color("#7EE787"),
	red:      lipgloss.Color("#FF5C7A"),
}

var (
	appStyle = lipgloss.NewStyle().Foreground(p.text).Background(p.bg)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(p.cyan).
			Background(p.panel).
			Padding(0, 1)

	pillStyle = lipgloss.NewStyle().
			Foreground(p.bg).
			Background(p.amber).
			Bold(true).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.border).
			Background(p.panel).
			Padding(0, 1)

	activePanelStyle = panelStyle.Copy().BorderForeground(p.cyan)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(p.amber).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().Foreground(p.muted)
	goodStyle  = lipgloss.NewStyle().Foreground(p.green).Bold(true)
	badStyle   = lipgloss.NewStyle().Foreground(p.red).Bold(true)
	infoStyle  = lipgloss.NewStyle().Foreground(p.cyan)

	selectedStationStyle = lipgloss.NewStyle().
				Foreground(p.bg).
				Background(p.cyan).
				Bold(true)

	stationStyle = lipgloss.NewStyle().Foreground(p.text)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.amber).
			Padding(0, 1).
			Background(p.panelAlt)

	footerStyle = lipgloss.NewStyle().
			Foreground(p.bg).
			Background(p.cyan).
			Bold(true).
			Padding(0, 1)
)

func New(
	ss *services.StationService,
	stats *services.StatisticsService,
	sys *services.SystemService,
	set *services.SettingsService,
	rb *services.RadioBrowserService,
	ns *services.NotificationService,
	player *player.AudioPlayer,
) *Model {
	input := textinput.New()
	input.Prompt = "❯ "
	input.Placeholder = "komut yazın: listele, cal tr-trt-fm, ses 70, help"
	input.PromptStyle = lipgloss.NewStyle().Foreground(p.amber).Bold(true)
	input.TextStyle = lipgloss.NewStyle().Foreground(p.text)
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(p.muted)
	input.CompletionStyle = lipgloss.NewStyle().Foreground(p.green)
	input.ShowSuggestions = true
	input.CharLimit = 240
	input.Width = 80

	m := &Model{
		commands:        make(map[string]shell.ShellCommand),
		stationService:  ss,
		statsService:    stats,
		systemService:   sys,
		settingsService: set,
		radioBrowser:    rb,
		notificationSvc: ns,
		player:          player,
		input:           input,
		output:          viewport.New(80, 12),
		historyCursor:   -1,
	}
	m.refreshStations()
	m.commandLog = []string{
		goodStyle.Render("Radio Terminal hazır."),
		"Sol panelden istasyonları görebilir, alttaki komut satırından tüm komutları çalıştırabilirsiniz.",
		"Enter: komutu çalıştırır, boşken seçili istasyonu çalar. Ctrl+S durdurur, Ctrl+L çıktıyı temizler, Ctrl+C çıkar.",
	}
	m.refreshOutput()
	return m
}

func Run(
	ss *services.StationService,
	stats *services.StatisticsService,
	sys *services.SystemService,
	set *services.SettingsService,
	rb *services.RadioBrowserService,
	ns *services.NotificationService,
	player *player.AudioPlayer,
) error {
	m := New(ss, stats, sys, set, rb, ns, player)
	shell.RegisterAllCommands(m, ss, stats, sys, set, rb, ns, player)
	m.refreshSuggestions()
	restoreOutput := ui.SetOutput(io.Discard)
	defer restoreOutput()
	defer m.fireExit()

	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func (m *Model) Register(name string, f shell.CommandFunc, desc, category string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands[name] = shell.ShellCommand{
		Name:     name,
		Func:     f,
		Desc:     desc,
		Category: category,
	}
}

func (m *Model) SetOnExit(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onExit = f
}

func (m *Model) UpdateLastList(list []models.RadioStation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastList = append([]models.RadioStation(nil), list...)
}

func (m *Model) GetLastList() []models.RadioStation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]models.RadioStation(nil), m.lastList...)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), tick())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tickMsg:
		m.refreshStations()
		m.refreshSuggestions()
		return m, tick()
	case commandResultMsg:
		m.busy = false
		m.busyCommand = ""
		m.appendCommandOutput(msg.line, msg.output, msg.err)
		m.refreshStations()
		m.refreshSuggestions()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab":
			if m.completeInput() {
				return m, nil
			}
			return m, m.updateInput(msg)
		case "enter":
			line := strings.TrimSpace(m.input.Value())
			if line == "" {
				return m, m.playSelectedCommand()
			}
			m.input.SetValue("")
			m.historyCursor = -1
			m.commandHist = append(m.commandHist, line)
			return m.handleCommandLine(line)
		case "ctrl+s":
			return m.handleCommandLine("dur")
		case "ctrl+l":
			m.commandLog = nil
			m.refreshOutput()
			return m, nil
		case "ctrl+r":
			m.refreshStations()
			m.message = "İstasyon listesi yenilendi."
			return m, nil
		case "up":
			if strings.TrimSpace(m.input.Value()) == "" {
				m.moveSelection(-1)
				return m, nil
			}
			if m.hasInputSuggestions() {
				return m, m.updateInput(msg)
			}
			m.recallHistory(-1)
			m.refreshSuggestions()
			return m, nil
		case "down":
			if strings.TrimSpace(m.input.Value()) == "" {
				m.moveSelection(1)
				return m, nil
			}
			if m.hasInputSuggestions() {
				return m, m.updateInput(msg)
			}
			m.recallHistory(1)
			m.refreshSuggestions()
			return m, nil
		case "pgup":
			m.output.PageUp()
			return m, nil
		case "pgdown":
			m.output.PageDown()
			return m, nil
		case "ctrl+f":
			if len(m.stations) > 0 {
				return m.handleCommandLine(fmt.Sprintf("favori %s", m.stations[m.selected].ID))
			}
		case "ctrl+n":
			if strings.TrimSpace(m.input.Value()) != "" {
				return m, m.updateInput(msg)
			}
			m.moveSelection(1)
			return m, nil
		case "ctrl+p":
			if strings.TrimSpace(m.input.Value()) != "" {
				return m, m.updateInput(msg)
			}
			m.moveSelection(-1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	var vp viewport.Model
	vp, cmd = m.output.Update(msg)
	m.output = vp
	cmds = append(cmds, cmd)

	m.refreshSuggestions()
	return m, tea.Batch(cmds...)
}

func (m *Model) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshSuggestions()
	return cmd
}

func (m *Model) hasInputSuggestions() bool {
	return strings.TrimSpace(m.input.Value()) != "" && len(m.input.MatchedSuggestions()) > 0
}

func (m *Model) completeInput() bool {
	currentValue := m.input.Value()
	if len(m.completionSet) > 1 && m.completionIdx >= 0 && m.completionIdx < len(m.completionSet) && currentValue == m.completionSet[m.completionIdx] {
		m.completionIdx = (m.completionIdx + 1) % len(m.completionSet)
		m.input.SetValue(m.completionSet[m.completionIdx])
		m.input.CursorEnd()
		m.refreshSuggestions()
		return true
	}

	if current := m.input.CurrentSuggestion(); current != "" {
		m.completionSet = uniqueStrings(m.input.MatchedSuggestions())
		m.completionIdx = indexOfString(m.completionSet, current)
		m.input.SetValue(current)
		m.input.CursorEnd()
		m.refreshSuggestions()
		return true
	}

	suggestions := uniqueStrings(m.completionSuggestions(m.input.Value()))
	if len(suggestions) == 0 {
		m.completionSet = nil
		m.completionIdx = 0
		return false
	}
	m.completionSet = suggestions
	m.completionIdx = 0
	m.input.SetValue(suggestions[0])
	m.input.CursorEnd()
	m.refreshSuggestions()
	return true
}

func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Radio Terminal başlatılıyor..."
	}

	header := m.renderHeader()
	input := m.renderInput()
	footer := m.renderFooter()
	layout := m.calculateLayout(lipgloss.Height(header), lipgloss.Height(input), lipgloss.Height(footer))
	m.applyLayout(layout)

	body := m.renderBody(layout)

	view := lipgloss.JoinVertical(lipgloss.Left, header, body, input, footer)
	return strings.TrimRight(appStyle.Width(m.width).Height(m.height).Render(view), "\n")
}

func (m *Model) handleCommandLine(line string) (tea.Model, tea.Cmd) {
	if m.busy {
		m.appendCommandOutput(line, "", errors.New("başka bir komut zaten çalışıyor, lütfen bekleyin"))
		return m, nil
	}

	cmdName := strings.ToLower(firstWord(line))
	switch cmdName {
	case "exit", "q", "quit":
		return m, tea.Quit
	case "help", "?":
		m.appendCommandOutput(line, m.helpText(), nil)
		return m, nil
	case "clear", "temizle":
		m.commandLog = nil
		m.refreshOutput()
		return m, nil
	}

	m.busy = true
	m.busyCommand = line
	return m, m.runCommand(line)
}

func (m *Model) runCommand(line string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.executeCommand(line)
		return commandResultMsg{line: line, output: output, err: err}
	}
}

func (m *Model) playSelectedCommand() tea.Cmd {
	if len(m.stations) == 0 {
		return func() tea.Msg {
			return commandResultMsg{line: "cal", output: "", err: fmt.Errorf("çalınacak istasyon yok")}
		}
	}
	line := fmt.Sprintf("cal %s", m.stations[m.selected].ID)
	m.commandHist = append(m.commandHist, line)
	m.busy = true
	m.busyCommand = line
	return m.runCommand(line)
}

func (m *Model) executeCommand(line string) (string, error) {
	parts, err := shellwords.Parse(line)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}
	if len(parts) == 0 {
		return "", nil
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	m.mu.RLock()
	cmd, ok := m.commands[cmdName]
	m.mu.RUnlock()
	if !ok {
		return "", errors.New(services.L.Get("unknown_command", map[string]interface{}{"cmd": cmdName}))
	}

	var buf bytes.Buffer
	ui.WithOutputAndWidth(&buf, m.commandOutputWidth(), func() {
		defer func() {
			if r := recover(); r != nil {
				ui.PrintError(services.L.Get("error_executing", map[string]interface{}{"error": r}))
			}
		}()
		cmd.Func(args)
	})
	return strings.TrimRight(buf.String(), "\n"), nil
}

func (m *Model) appendCommandOutput(line, output string, err error) {
	prompt := infoStyle.Render("❯ " + line)
	m.commandLog = append(m.commandLog, prompt)
	if err != nil {
		m.commandLog = append(m.commandLog, badStyle.Render("  ✘ "+err.Error()))
	} else if strings.TrimSpace(output) != "" {
		m.commandLog = append(m.commandLog, normalizeCommandOutput(output, m.commandOutputWidth())...)
	}
	if len(m.commandLog) > 300 {
		m.commandLog = m.commandLog[len(m.commandLog)-300:]
	}
	m.refreshOutput()
}

func (m *Model) refreshOutput() {
	width := m.commandOutputWidth()
	lines := make([]string, 0, len(m.commandLog))
	for _, line := range m.commandLog {
		lines = append(lines, fitOutputLine(line, width))
	}
	m.output.SetContent(strings.Join(lines, "\n"))
	m.output.GotoBottom()
}

func (m *Model) refreshStations() {
	stations := m.stationService.GetAllStations()
	if len(stations) == 0 {
		m.stations = nil
		m.selected = 0
		return
	}

	m.mu.RLock()
	lastList := append([]models.RadioStation(nil), m.lastList...)
	m.mu.RUnlock()

	if len(lastList) > 0 {
		known := make(map[string]models.RadioStation, len(stations))
		for _, st := range stations {
			known[st.ID] = st
		}
		filtered := make([]models.RadioStation, 0, len(lastList))
		for _, st := range lastList {
			if fresh, ok := known[st.ID]; ok {
				filtered = append(filtered, fresh)
			}
		}
		if len(filtered) > 0 {
			m.stations = filtered
			m.selected = clamp(m.selected, 0, len(m.stations)-1)
			return
		}
	}

	m.stations = stations
	m.selected = clamp(m.selected, 0, len(m.stations)-1)
}

func (m *Model) refreshSuggestions() {
	suggestions := m.completionSuggestions(m.input.Value())
	m.input.SetSuggestions(uniqueStrings(suggestions))
}

func (m *Model) completionSuggestions(current string) []string {
	words := completionWords(current)
	var suggestions []string
	tokenStart := completionTokenStart(current)
	token := current[tokenStart:]
	prefix := current[:tokenStart]

	if len(words) <= 1 && !strings.HasSuffix(current, " ") {
		for name := range m.commands {
			suggestions = append(suggestions, name)
		}
		suggestions = append(suggestions, "help", "exit")
		return buildCompletionSuggestions(prefix, token, suggestions)
	} else if len(words) > 0 {
		cmd := strings.ToLower(words[0])
		if strings.HasPrefix(token, "-") {
			return buildCompletionSuggestions(prefix, token, shell.FlagSuggestions(cmd))
		}
		switch cmd {
		case "cal", "kontrol", "favori", "sil", "duzenle":
			return buildStationCompletionSuggestions(prefix, token, m.stationService.GetAllStations())
		case "ulke":
			suggestions = append(suggestions, m.stationService.GetCountries()...)
		case "tur":
			suggestions = append(suggestions, m.stationService.GetGenres()...)
		case "tema":
			suggestions = append(suggestions, ui.GetThemes()...)
		case "dil", "lang":
			for code := range services.L.GetLanguages() {
				suggestions = append(suggestions, code)
			}
		}
	}
	return buildCompletionSuggestions(prefix, token, suggestions)
}

func completionWords(text string) []string {
	words := strings.Fields(text)
	if strings.HasSuffix(text, " ") {
		words = append(words, "")
	}
	return words
}

func completionTokenStart(text string) int {
	idx := strings.LastIndexFunc(text, unicode.IsSpace)
	if idx < 0 {
		return 0
	}
	return idx + 1
}

func buildCompletionSuggestions(prefix, current string, values []string) []string {
	values = append([]string(nil), values...)
	sort.Strings(values)

	currentLower := strings.ToLower(current)
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		if strings.HasPrefix(strings.ToLower(value), currentLower) {
			out = append(out, prefix+value+" ")
		}
	}
	return out
}

func buildStationCompletionSuggestions(prefix, current string, stations []models.RadioStation) []string {
	currentNorm := normalizeCompletionText(current)
	seen := make(map[string]struct{}, len(stations))
	out := make([]string, 0, len(stations))

	add := func(st models.RadioStation) {
		if st.ID == "" {
			return
		}
		if _, ok := seen[st.ID]; ok {
			return
		}
		seen[st.ID] = struct{}{}
		out = append(out, prefix+st.ID+" ")
	}
	matchesIDPrefix := func(st models.RadioStation) bool {
		return strings.HasPrefix(normalizeCompletionText(st.ID), currentNorm)
	}
	matchesNamePrefix := func(st models.RadioStation) bool {
		return strings.HasPrefix(normalizeCompletionText(st.Name), currentNorm)
	}
	matchesIDContains := func(st models.RadioStation) bool {
		return strings.Contains(normalizeCompletionText(st.ID), currentNorm)
	}
	matchesNameContains := func(st models.RadioStation) bool {
		return strings.Contains(normalizeCompletionText(st.Name), currentNorm)
	}

	if currentNorm == "" {
		for _, st := range stations {
			add(st)
		}
		return out
	}

	for _, st := range stations {
		if matchesIDPrefix(st) {
			add(st)
		}
	}
	for _, st := range stations {
		if matchesNamePrefix(st) {
			add(st)
		}
	}
	for _, st := range stations {
		if matchesIDContains(st) {
			add(st)
		}
	}
	for _, st := range stations {
		if matchesNameContains(st) {
			add(st)
		}
	}
	return out
}

func normalizeCompletionText(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"ç", "c",
		"ğ", "g",
		"ı", "i",
		"ö", "o",
		"ş", "s",
		"ü", "u",
	)
	return replacer.Replace(s)
}

func (m *Model) recallHistory(delta int) {
	if len(m.commandHist) == 0 {
		return
	}
	if m.historyCursor < 0 {
		m.historyCursor = len(m.commandHist)
	}
	m.historyCursor = clamp(m.historyCursor+delta, 0, len(m.commandHist))
	if m.historyCursor == len(m.commandHist) {
		m.input.SetValue("")
		return
	}
	m.input.SetValue(m.commandHist[m.historyCursor])
}

func (m *Model) moveSelection(delta int) {
	if len(m.stations) == 0 {
		return
	}
	m.selected = (m.selected + delta) % len(m.stations)
	if m.selected < 0 {
		m.selected += len(m.stations)
	}
}

func (m *Model) resize() {
	inputWidth := max(8, m.width-8)
	m.input.Width = inputWidth

	layout := m.calculateLayout(1, 3, 1)
	m.applyLayout(layout)
	m.refreshOutput()
}

func (m *Model) calculateLayout(headerHeight, inputHeight, footerHeight int) screenLayout {
	bodyHeight := m.height - headerHeight - inputHeight - footerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	layout := screenLayout{bodyHeight: bodyHeight}
	if m.width >= 72 && bodyHeight >= 8 {
		gap := 2
		leftWidth := clamp(int(float64(m.width)*0.36), 24, 42)
		rightWidth := m.width - leftWidth - gap
		if rightWidth < 34 {
			rightWidth = 34
			leftWidth = m.width - rightWidth - gap
		}
		if leftWidth >= 20 && rightWidth >= 30 && leftWidth+rightWidth+gap <= m.width {
			layout.sideBySide = true
			layout.leftWidth = leftWidth
			layout.rightWidth = rightWidth
			layout.stationHeight = bodyHeight
			layout.rightHeight = bodyHeight
		}
	}

	if !layout.sideBySide {
		layout.leftWidth = m.width
		layout.rightWidth = m.width
		if bodyHeight >= 14 {
			layout.stationHeight = clamp(bodyHeight/2, 6, 10)
			layout.rightHeight = bodyHeight - layout.stationHeight - 1
			if layout.rightHeight < 5 {
				layout.rightHeight = 5
				layout.stationHeight = max(0, bodyHeight-layout.rightHeight-1)
			}
		} else {
			layout.stationHeight = 0
			layout.rightHeight = bodyHeight
		}
	}

	if layout.rightHeight < 1 {
		layout.rightHeight = 1
	}
	layout.outputViewportWidth, layout.outputViewportHeight = outputViewportSize(layout.rightWidth, layout.rightHeight)
	return layout
}

func (m *Model) applyLayout(layout screenLayout) {
	inputWidth := max(8, m.width-8)
	if m.input.Width != inputWidth {
		m.input.Width = inputWidth
	}
	if layout.outputViewportWidth > 0 && m.output.Width != layout.outputViewportWidth {
		m.output.Width = layout.outputViewportWidth
		m.refreshOutput()
	}
	if layout.outputViewportHeight > 0 && m.output.Height != layout.outputViewportHeight {
		m.output.Height = layout.outputViewportHeight
	}
}

func (m *Model) renderBody(layout screenLayout) string {
	if layout.bodyHeight <= 0 {
		return ""
	}

	if layout.sideBySide {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderStationPanel(layout.leftWidth, layout.stationHeight),
			"  ",
			m.renderRightPanel(layout.rightWidth, layout.rightHeight),
		)
	}

	parts := make([]string, 0, 3)
	if layout.stationHeight > 0 {
		parts = append(parts, m.renderStationPanel(layout.leftWidth, layout.stationHeight))
		if layout.stationHeight+layout.rightHeight < layout.bodyHeight {
			parts = append(parts, "")
		}
	}
	parts = append(parts, m.renderRightPanel(layout.rightWidth, layout.rightHeight))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *Model) renderHeader() string {
	station, song, vol, muted, playing, recording, elapsed := m.player.GetStatus()
	left := titleStyle.Render("RADIO TERMINAL")
	mode := pillStyle.Render("TUI")
	stateText := services.L.Get("stopped")
	stateStyle := mutedStyle
	if playing && station != nil {
		stateText = station.Name
		if song != "" {
			stateText += " · " + song
		}
		stateStyle = goodStyle
	}
	rec := ""
	if recording {
		rec = " · REC"
	}
	volume := fmt.Sprintf("Ses %d%%", vol)
	if muted {
		volume = services.L.Get("muted") + " · " + volume
	}
	rightText := fmt.Sprintf("%s · %s%s", volume, formatElapsed(elapsed), rec)

	if m.width < 34 {
		plain := "RADIO TERMINAL · " + stateText
		return lipgloss.NewStyle().Width(m.width).Background(p.panel).Render(fitOutputLine(infoStyle.Render(plain), m.width))
	}

	prefix := lipgloss.JoinHorizontal(lipgloss.Center, left, " ", mode)
	rightMax := max(0, min(lipgloss.Width(rightText), m.width/3))
	right := mutedStyle.Render(truncate(rightText, rightMax))
	stateMax := m.width - lipgloss.Width(prefix) - lipgloss.Width(right) - 3
	if stateMax < 8 {
		right = ""
		stateMax = m.width - lipgloss.Width(prefix) - 2
	}
	if stateMax < 1 {
		return lipgloss.NewStyle().Width(m.width).Background(p.panel).Render(fitOutputLine(prefix, m.width))
	}

	state := stateStyle.Render(truncate(stateText, stateMax))
	line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, " ", state)
	padding := max(0, m.width-lipgloss.Width(line)-lipgloss.Width(right))
	content := line
	if right != "" {
		content += strings.Repeat(" ", padding) + right
	}
	return lipgloss.NewStyle().Width(m.width).Background(p.panel).Render(fitOutputLine(content, m.width))
}

func (m *Model) renderStationPanel(width, height int) string {
	innerWidth := max(10, width-4)
	innerHeight := max(4, height-2)
	title := sectionTitleStyle.Render("İstasyonlar")
	lastListLen := m.lastListLen()
	if lastListLen > 0 {
		title += " " + mutedStyle.Render(fmt.Sprintf("(%d filtreli)", len(m.stations)))
	} else {
		title += " " + mutedStyle.Render(fmt.Sprintf("(%d)", len(m.stations)))
	}

	listHeight := innerHeight - 2
	if listHeight < 1 {
		listHeight = 1
	}

	// Adjust sliding window scrollStart
	if m.selected < m.scrollStart {
		m.scrollStart = m.selected
	} else if m.selected >= m.scrollStart+listHeight {
		m.scrollStart = m.selected - listHeight + 1
	}
	m.scrollStart = clamp(m.scrollStart, 0, max(0, len(m.stations)-listHeight))
	start := m.scrollStart

	lines := []string{
		fitOutputLine(title, innerWidth),
		mutedStyle.Render(truncate("↑/↓ seç · Enter çal · Ctrl+F favori", innerWidth)),
	}
	for i := 0; i < listHeight && start+i < len(m.stations); i++ {
		idx := start + i
		st := m.stations[idx]
		fav := " "
		if st.Favorite {
			fav = "★"
		}
		stName := padRight(truncate(st.Name, 28), 28)
		stCountry := truncate(firstNonEmpty(st.Country, "—"), 10)
		line := fmt.Sprintf("%s %2d %s %s", fav, idx+1, stName, stCountry)
		line = truncate(line, innerWidth)
		if idx == m.selected {
			lines = append(lines, selectedStationStyle.Width(innerWidth).Render(line))
		} else {
			lines = append(lines, stationStyle.Width(innerWidth).Render(line))
		}
	}
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	return activePanelStyle.Width(max(1, width-2)).Height(max(1, height-2)).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderRightPanel(width, height int) string {
	if height <= 0 {
		return ""
	}

	nowHeight, outputBoxHeight := rightPanelHeights(height)
	if nowHeight == 0 {
		return m.renderCommandOutputBox(width, outputBoxHeight)
	}

	now := m.renderNowPlaying(width, nowHeight)
	outputBox := m.renderCommandOutputBox(width, outputBoxHeight)
	return lipgloss.JoinVertical(lipgloss.Left, now, outputBox)
}

func (m *Model) renderCommandOutputBox(width, height int) string {
	if height <= 0 {
		return ""
	}
	contentHeight := max(1, height-2)
	title := sectionTitleStyle.Render("Komut Çıktısı")
	viewHeight := max(0, contentHeight-1)
	view := m.output.View()
	if viewHeight == 0 {
		view = ""
	}
	outputBox := panelStyle.Copy().
		Width(max(1, width-2)).
		Height(max(1, height-2)).
		BorderForeground(p.border).
		Render(title + "\n" + view)
	return outputBox
}

func (m *Model) renderNowPlaying(width, height int) string {
	station, song, vol, muted, playing, recording, elapsed := m.player.GetStatus()
	codec, bitrate, sampleRate := m.player.GetCodecInfo()
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	lines := []string{sectionTitleStyle.Render("Şimdi Çalıyor")}
	if !playing || station == nil {
		lines = append(lines, mutedStyle.Render(truncate("Radyo duruyor. Sol listeden istasyon seçip Enter'a basın.", innerWidth)))
	} else {
		lines = append(lines, goodStyle.Render(truncate(station.Name, innerWidth)))
		if song != "" && len(lines) < innerHeight-1 {
			lines = append(lines, infoStyle.Render(truncate(song, innerWidth)))
		} else if elapsed < 15 {
			lines = append(lines, mutedStyle.Render(services.L.Get("waiting_song")))
		}
		if len(lines) < innerHeight-1 {
			meta := []string{
				firstNonEmpty(station.Country, "—"),
				firstNonEmpty(station.Genre, "—"),
				firstNonEmpty(joinNonEmpty(" · ", codec, bitrate, sampleRate), "—"),
			}
			lines = append(lines, mutedStyle.Render(truncate(strings.Join(meta, "   "), innerWidth)))
		}
	}

	volume := fmt.Sprintf("Ses %d%%", vol)
	if muted {
		volume = services.L.Get("muted") + " · " + volume
	}
	flags := []string{volume, formatElapsed(elapsed)}
	if recording {
		flags = append(flags, "KAYIT")
	}
	if len(lines) < innerHeight {
		lines = append(lines, mutedStyle.Render(truncate(strings.Join(flags, "   "), innerWidth)))
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	return panelStyle.Copy().
		Width(max(1, width-2)).
		Height(max(1, height-2)).
		BorderForeground(p.amber).
		Render(strings.Join(lines, "\n"))
}

func (m *Model) renderInput() string {
	statusText := ""
	if m.busy {
		statusText = " çalışıyor: " + m.busyCommand
	} else if m.message != "" {
		statusText = " " + m.message
	}
	innerWidth := max(1, m.width-4)
	status := mutedStyle.Render(truncate(statusText, max(0, innerWidth-lipgloss.Width(m.input.View()))))
	content := fitOutputLine(m.input.View()+status, innerWidth)
	return inputStyle.Width(max(1, m.width-2)).Render(content)
}

func (m *Model) renderFooter() string {
	station, song, vol, muted, playing, recording, elapsed := m.player.GetStatus()
	codec, bitrate, sampleRate := m.player.GetCodecInfo()

	status := services.L.Get("stopped")
	if playing && station != nil {
		title := station.Name
		if song != "" {
			title += " · " + song
		} else if elapsed < 15 {
			title += " · " + services.L.Get("waiting_song")
		}
		parts := []string{
			title,
			firstNonEmpty(station.Country, "—"),
			firstNonEmpty(station.Genre, "—"),
			firstNonEmpty(joinNonEmpty(" ", codec, bitrate, sampleRate), "—"),
			fmt.Sprintf("Ses %d%%", vol),
			formatElapsed(elapsed),
		}
		if muted {
			parts[4] = services.L.Get("muted") + " " + parts[4]
		}
		if recording {
			parts = append(parts, services.L.Get("recording"))
		}
		status = strings.Join(parts, "  │  ")
	}

	return footerStyle.Width(m.width).Render(truncate(status, max(1, m.width-2)))
}

func (m *Model) helpText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	categories := make(map[string][]shell.ShellCommand)
	var catNames []string
	for _, cmd := range m.commands {
		if _, ok := categories[cmd.Category]; !ok {
			catNames = append(catNames, cmd.Category)
		}
		categories[cmd.Category] = append(categories[cmd.Category], cmd)
	}
	sort.Strings(catNames)

	var b strings.Builder
	for _, cat := range catNames {
		b.WriteString("\n")
		b.WriteString(sectionTitleStyle.Render(services.L.Get(cat)))
		b.WriteString("\n")
		cmds := categories[cat]
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
		for _, cmd := range cmds {
			b.WriteString(fmt.Sprintf("  %-18s %s\n", cmd.Name, services.L.Get(cmd.Desc)))
		}
	}
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(services.L.Get("cat_general")))
	b.WriteString("\n  help / ?          ")
	b.WriteString(services.L.Get("help_general"))
	b.WriteString("\n  exit / q / quit   ")
	b.WriteString(services.L.Get("exit_general"))
	return strings.TrimSpace(b.String())
}

func (m *Model) fireExit() {
	m.exitOnce.Do(func() {
		m.mu.RLock()
		onExit := m.onExit
		m.mu.RUnlock()
		if onExit != nil {
			onExit()
		}
	})
}

func (m *Model) lastListLen() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.lastList)
}

func (m *Model) commandOutputWidth() int {
	if m.output.Width > 4 {
		return m.output.Width
	}
	if m.width > 12 {
		return max(20, m.width/2-8)
	}
	return 78
}

func rightPanelHeights(height int) (int, int) {
	if height <= 0 {
		return 0, 0
	}
	if height < 9 {
		return 0, height
	}
	nowHeight := 7
	if height < 12 {
		nowHeight = 5
	}
	outputHeight := height - nowHeight
	if outputHeight < 4 {
		outputHeight = 4
		nowHeight = max(0, height-outputHeight)
	}
	return nowHeight, outputHeight
}

func outputViewportSize(width, rightHeight int) (int, int) {
	_, outputBoxHeight := rightPanelHeights(rightHeight)
	innerWidth := max(1, width-4)
	innerHeight := max(1, outputBoxHeight-2)
	return innerWidth, max(1, innerHeight-1)
}

func normalizeCommandOutput(output string, width int) []string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	rawLines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimRight(line, " \t")
		lines = append(lines, fitOutputLine(line, width))
	}
	return lines
}

func fitOutputLine(line string, width int) string {
	if width <= 0 {
		width = 78
	}
	if ansi.StringWidth(line) <= width {
		return line
	}
	return ansi.Truncate(line, width, "…")
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func firstWord(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func joinNonEmpty(sep string, values ...string) string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, sep)
}

func formatElapsed(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	ellipsis := "…"
	target := max(0, width-lipgloss.Width(ellipsis))
	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next) > target {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + ellipsis
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func indexOfString(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return 0
}

func clamp(v, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
