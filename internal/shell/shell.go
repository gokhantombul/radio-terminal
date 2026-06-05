package shell

import (
	"fmt"
	"radio-shell/internal/models"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/ui"
	"sort"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
)

type CommandFunc func(args []string)

type ShellCommand struct {
	Name     string
	Func     CommandFunc
	Desc     string
	Category string
}

type InteractiveShell struct {
	commands       map[string]ShellCommand
	stationService *services.StationService
	player         *player.AudioPlayer
	running        bool
	lastList       []models.RadioStation
	onExit         func()
}

func NewInteractiveShell(ss *services.StationService, p *player.AudioPlayer) *InteractiveShell {
	return &InteractiveShell{
		commands:       make(map[string]ShellCommand),
		stationService: ss,
		player:         p,
		running:        true,
	}
}

func (s *InteractiveShell) Register(name string, f CommandFunc, desc, category string) {
	s.commands[name] = ShellCommand{
		Name:     name,
		Func:     f,
		Desc:     desc,
		Category: category,
	}
}

func (s *InteractiveShell) SetOnExit(f func()) {
	s.onExit = f
}

func (s *InteractiveShell) Run() {
	defer func() {
		if s.onExit != nil {
			s.onExit()
		}
	}()

	ui.PrintBanner()
	fmt.Printf("          %s\n\n", services.L.Get("welcome_msg"))

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          s.getPrompt(),
		HistoryFile:     "/tmp/radio-shell.history",
		AutoComplete:    newRadioCompleter(s),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for s.running {
		rl.SetPrompt(s.getPrompt())
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts, err := shellwords.Parse(line)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Parse error: %v", err))
			continue
		}

		if len(parts) == 0 {
			continue
		}

		cmdName := strings.ToLower(parts[0])
		args := parts[1:]

		if cmdName == "exit" || cmdName == "q" || cmdName == "quit" {
			s.running = false
			break
		}

		if cmdName == "help" || cmdName == "?" {
			s.printHelp()
			continue
		}

		if cmd, ok := s.commands[cmdName]; ok {
			func() {
				defer func() {
					if r := recover(); r != nil {
						ui.PrintError(services.L.Get("error_executing", map[string]interface{}{"error": r}))
					}
				}()
				cmd.Func(args)
			}()
		} else {
			ui.PrintError(services.L.Get("unknown_command", map[string]interface{}{"cmd": cmdName}))
		}
	}
}

func (s *InteractiveShell) getPrompt() string {
	station, song, _, _, playing, _, _ := s.player.GetStatus()

	p := ""
	if ui.CurrentThemeName == "winamp-classic" {
		p = ui.CurrentTheme.Primary.Sprint("▌▌ ")
	} else {
		p = "📻 "
	}

	if playing && station != nil {
		name := station.Name
		if song != "" {
			name = fmt.Sprintf("%s (%s)", name, song)
		}
		p += ui.CurrentTheme.Primary.Sprint(name)
	} else {
		p += ui.CurrentTheme.Primary.Sprint("radio")
	}

	p += colorHighlight(" ❯ ")
	return p
}

func colorHighlight(s string) string {
	return ui.CurrentTheme.Highlight.Sprint(s)
}

func (s *InteractiveShell) printHelp() {
	ui.PrintHeader(services.L.Get("help_title"))

	categories := make(map[string][]ShellCommand)
	var catNames []string
	for _, cmd := range s.commands {
		if _, ok := categories[cmd.Category]; !ok {
			catNames = append(catNames, cmd.Category)
		}
		categories[cmd.Category] = append(categories[cmd.Category], cmd)
	}
	sort.Strings(catNames)

	for _, cat := range catNames {
		ui.CurrentTheme.Primary.Printf("\n  %s\n", services.L.Get(cat))
		cmds := categories[cat]
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
		for _, cmd := range cmds {
			ui.CurrentTheme.Primary.Printf("    %-20s", cmd.Name)
			fmt.Printf(" - %s\n", services.L.Get(cmd.Desc))
		}
	}

	ui.CurrentTheme.Primary.Printf("\n  %s\n", services.L.Get("cat_general"))
	ui.CurrentTheme.Primary.Printf("    %-20s", "help / ?")
	fmt.Printf(" - %s\n", services.L.Get("help_general"))
	ui.CurrentTheme.Primary.Printf("    %-20s", "exit / q / quit")
	fmt.Printf(" - %s\n", services.L.Get("exit_general"))
}

func (s *InteractiveShell) UpdateLastList(list []models.RadioStation) {
	s.lastList = list
}

func (s *InteractiveShell) GetLastList() []models.RadioStation {
	return s.lastList
}

type radioCompleter struct {
	shell *InteractiveShell
}

func newRadioCompleter(shell *InteractiveShell) readline.AutoCompleter {
	return &radioCompleter{shell: shell}
}

func (c *radioCompleter) Do(line []rune, pos int) ([][]rune, int) {
	if pos > len(line) {
		pos = len(line)
	}
	text := string(line[:pos])
	words := completionWords(text)
	if len(words) <= 1 {
		values := make([]string, 0, len(c.shell.commands)+4)
		for name := range c.shell.commands {
			values = append(values, name)
		}
		values = append(values, "help", "exit", "q", "?")
		current := ""
		if len(words) == 1 {
			current = words[0]
		}
		return completionCandidates(current, values)
	}

	cmd := strings.ToLower(words[0])
	current := words[len(words)-1]
	if strings.HasPrefix(current, "-") {
		return completionCandidates(current, FlagSuggestions(cmd))
	}

	switch cmd {
	case "cal", "kontrol", "favori", "sil", "duzenle":
		var values []string
		for _, st := range c.shell.stationService.GetAllStations() {
			values = append(values, st.ID)
		}
		return completionCandidates(current, values)
	case "ulke":
		return completionCandidates(current, c.shell.stationService.GetCountries())
	case "tur":
		return completionCandidates(current, c.shell.stationService.GetGenres())
	case "tema":
		return completionCandidates(current, ui.GetThemes())
	case "dil", "lang":
		var values []string
		for code := range services.L.GetLanguages() {
			values = append(values, code)
		}
		return completionCandidates(current, values)
	}

	return nil, 0
}

func completionWords(text string) []string {
	words := strings.Fields(text)
	if strings.HasSuffix(text, " ") {
		words = append(words, "")
	}
	return words
}

func completionCandidates(current string, values []string) ([][]rune, int) {
	sort.Strings(values)
	offset := len([]rune(current))
	currentLower := strings.ToLower(current)
	seen := make(map[string]struct{}, len(values))
	var out [][]rune
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		if strings.HasPrefix(strings.ToLower(value), currentLower) {
			valueRunes := []rune(value)
			if offset > len(valueRunes) {
				continue
			}
			suffix := append([]rune{}, valueRunes[offset:]...)
			suffix = append(suffix, ' ')
			out = append(out, suffix)
		}
	}
	return out, offset
}

func FlagSuggestions(cmd string) []string {
	switch cmd {
	case "ulke", "tur", "cal", "favori", "kontrol", "dil", "lang":
		return []string{"-i"}
	case "ara", "ses":
		return []string{"-s"}
	case "karistir", "rastgele":
		return []string{"-u", "-t"}
	case "uyku":
		return []string{"-d"}
	case "iceaktar":
		return []string{"-d", "-u", "-t", "-p"}
	case "online-ara":
		return []string{"-s", "-u", "-t", "-l"}
	case "online-ekle":
		return []string{"-n"}
	}
	return nil
}
