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

func (s *InteractiveShell) Run() {
	ui.PrintBanner()
	fmt.Printf("          %s\n\n", services.L.Get("welcome_msg"))

	// Build completer
	var items []readline.PrefixCompleterInterface
	for name := range s.commands {
		items = append(items, readline.PcItem(name))
	}
	items = append(items, readline.PcItem("help"), readline.PcItem("exit"), readline.PcItem("q"))

	completer := readline.NewPrefixCompleter(items...)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          s.getPrompt(),
		HistoryFile:     "/tmp/radio-shell.history",
		AutoComplete:    completer,
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
