package ui

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"radio-shell/internal/models"
	"radio-shell/internal/services"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/rodaine/table"
)

type Theme struct {
	Primary   *color.Color
	Secondary *color.Color
	Highlight *color.Color
	Success   *color.Color
	Error     *color.Color
}

var Themes = map[string]Theme{
	"default": {
		Primary:   color.New(color.FgCyan, color.Bold),
		Secondary: color.New(color.FgBlue),
		Highlight: color.New(color.FgMagenta, color.Bold),
		Success:   color.New(color.FgGreen),
		Error:     color.New(color.FgRed, color.Bold),
	},
	"hacker": {
		Primary:   color.New(color.FgHiGreen, color.Bold),
		Secondary: color.New(color.FgGreen),
		Highlight: color.New(color.FgHiGreen),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed),
	},
	"ocean": {
		Primary:   color.New(color.FgHiBlue, color.Bold),
		Secondary: color.New(color.FgCyan),
		Highlight: color.New(color.FgHiWhite, color.Bold),
		Success:   color.New(color.FgHiCyan),
		Error:     color.New(color.FgHiRed),
	},
	"sunset": {
		Primary:   color.New(color.FgHiYellow, color.Bold),
		Secondary: color.New(color.FgHiRed),
		Highlight: color.New(color.FgHiMagenta, color.Bold),
		Success:   color.New(color.FgHiYellow),
		Error:     color.New(color.FgHiRed, color.Bold),
	},
	"midnight": {
		Primary:   color.New(color.FgHiMagenta, color.Bold),
		Secondary: color.New(color.FgBlue),
		Highlight: color.New(color.FgHiCyan, color.Bold),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed),
	},
	"sakura": {
		Primary:   color.New(color.FgHiMagenta),
		Secondary: color.New(color.FgMagenta),
		Highlight: color.New(color.FgHiWhite, color.Bold),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed),
	},
	"winamp-classic": {
		Primary:   color.New(color.FgGreen),
		Secondary: color.New(color.FgBlue),
		Highlight: color.New(color.FgYellow),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed),
	},
	"besiktas": {
		Primary:   color.New(color.FgHiWhite, color.Bold),
		Secondary: color.New(color.FgHiBlack),
		Highlight: color.New(color.FgHiRed, color.Bold),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed, color.Bold),
	},
}

var (
	CurrentTheme               = Themes["default"]
	CurrentThemeName           = "default"
	Output           io.Writer = os.Stdout
	OutputWidth      int       // 0 = terminal auto, >0 = TUI fixed width
	outputMu         sync.Mutex
)

func WithOutput(w io.Writer, f func()) {
	restore := SetOutput(w)
	defer restore()
	f()
}

// WithOutputAndWidth temporarily redirects all UI output to w and sets OutputWidth.
// Used by the TUI to capture command output at a known viewport width.
func WithOutputAndWidth(w io.Writer, width int, f func()) {
	if w == nil {
		w = os.Stdout
	}
	outputMu.Lock()
	oldOut := Output
	oldColor := color.Output
	oldWidth := OutputWidth
	Output = w
	color.Output = w
	OutputWidth = width
	outputMu.Unlock()
	defer func() {
		outputMu.Lock()
		Output = oldOut
		color.Output = oldColor
		OutputWidth = oldWidth
		outputMu.Unlock()
	}()
	f()
}

func SetOutput(w io.Writer) func() {
	if w == nil {
		w = os.Stdout
	}
	outputMu.Lock()
	old := Output
	oldColorOutput := color.Output
	Output = w
	color.Output = w
	outputMu.Unlock()

	return func() {
		outputMu.Lock()
		Output = old
		color.Output = oldColorOutput
		outputMu.Unlock()
	}
}

func Fprint(a ...interface{}) {
	fmt.Fprint(Output, a...)
}

func Fprintf(format string, a ...interface{}) {
	fmt.Fprintf(Output, format, a...)
}

func Fprintln(a ...interface{}) {
	fmt.Fprintln(Output, a...)
}

func LoadTheme() {
	home, _ := os.UserHomeDir()
	themeFile := filepath.Join(home, ".radio-shell", "theme")
	if data, err := ioutil.ReadFile(themeFile); err == nil {
		name := strings.TrimSpace(string(data))
		if t, ok := Themes[name]; ok {
			CurrentTheme = t
			CurrentThemeName = name
		}
	}
}

func SetTheme(name string) bool {
	if t, ok := Themes[name]; ok {
		CurrentTheme = t
		CurrentThemeName = name
		home, _ := os.UserHomeDir()
		themeFile := filepath.Join(home, ".radio-shell", "theme")
		os.MkdirAll(filepath.Dir(themeFile), 0755)
		ioutil.WriteFile(themeFile, []byte(name), 0644)
		return true
	}
	return false
}

func GetThemes() []string {
	names := make([]string, 0, len(Themes))
	for name := range Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func IsCurrentTheme(name string) bool {
	return CurrentThemeName == name
}

func PrintBanner() {
	width := 66
	appTitle := services.L.Get("app_title")

	fmt.Fprintln(Output)
	CurrentTheme.Secondary.Fprintf(Output, "  ╔%s╗\n", strings.Repeat("═", width))
	CurrentTheme.Secondary.Fprint(Output, "  ║")
	fmt.Fprint(Output, "                  ♬  ")
	CurrentTheme.Highlight.Fprint(Output, "░░░ RADIO SHELL ░░░")
	fmt.Fprint(Output, "  ♬                   ")
	CurrentTheme.Secondary.Fprintln(Output, "║")

	CurrentTheme.Secondary.Fprint(Output, "  ║")
	padding := (width - len(appTitle)) / 2
	fmt.Fprint(Output, strings.Repeat(" ", padding))
	CurrentTheme.Primary.Fprint(Output, appTitle)
	fmt.Fprint(Output, strings.Repeat(" ", width-padding-len(appTitle)))
	CurrentTheme.Secondary.Fprintln(Output, "║")

	CurrentTheme.Secondary.Fprint(Output, "  ║")
	version := "v3.0.0 | Go 1.25 + Fatih Color"
	paddingVer := (width - len(version)) / 2
	fmt.Fprint(Output, strings.Repeat(" ", paddingVer))
	fmt.Fprint(Output, version)
	fmt.Fprint(Output, strings.Repeat(" ", width-paddingVer-len(version)))
	CurrentTheme.Secondary.Fprintln(Output, "║")

	CurrentTheme.Secondary.Fprintf(Output, "  ╚%s╝\n", strings.Repeat("═", width))
	fmt.Fprintln(Output)
}

func PrintHeader(title string) {
	if OutputWidth > 0 {
		// Compact header for TUI viewport — no extra blank lines
		CurrentTheme.Highlight.Fprint(Output, "❯❯ ")
		CurrentTheme.Primary.Fprint(Output, strings.ToUpper(title))
		CurrentTheme.Highlight.Fprintln(Output, " ❮❮")
		return
	}
	fmt.Fprintln(Output)
	CurrentTheme.Highlight.Fprint(Output, " ❯❯ ")
	CurrentTheme.Primary.Fprint(Output, strings.ToUpper(title))
	CurrentTheme.Highlight.Fprintln(Output, " ❮❮ ")
	fmt.Fprintln(Output, strings.Repeat("─", 40))
}

func PrintError(msg string) {
	CurrentTheme.Error.Fprintf(Output, "  ✘ %s\n", msg)
}

func PrintSuccess(msg string) {
	CurrentTheme.Success.Fprintf(Output, "  ✔ %s\n", msg)
}

func PrintInfo(msg string) {
	CurrentTheme.Highlight.Fprintf(Output, "  ℹ %s\n", msg)
}

func PrintStationTable(title string, stations []models.RadioStation, subtitle string) {
	if len(stations) == 0 {
		PrintInfo(services.L.Get("no_stations"))
		return
	}

	PrintHeader(title)

	if OutputWidth > 0 {
		// Compact single-line format for TUI viewport.
		// Priority: fav ★ · index · name · country · genre
		// Name gets the most space; country and genre fill the rest.
		avail := OutputWidth - 2 // 2 chars leading indent
		nameW := 24
		cntryW := 10
		if avail < 55 {
			nameW = 16
			cntryW = 8
		}
		for i, s := range stations {
			fav := " "
			if s.Favorite {
				fav = "★"
			}
			name := runesTruncate(s.Name, nameW)
			country := runesTruncate(firstNonEmptyStr(s.Country, "—"), cntryW)
			genre := s.Genre
			// Build: "  ★  25  TRT FM                  Türkiye     Pop/Türkçe"
			line := fmt.Sprintf("  %s %3d  %-*s  %-*s  %s",
				fav, i+1,
				nameW, name,
				cntryW, country,
				genre)
			if s.Favorite {
				CurrentTheme.Highlight.Fprintln(Output, line)
			} else {
				fmt.Fprintln(Output, line)
			}
		}
	} else {
		tbl := table.New("NO", "ID", "STATION NAME", "COUNTRY", "GENRE", "FAV").WithWriter(Output)
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return CurrentTheme.Highlight.Sprintf(strings.ToUpper(format), vals...)
		})
		for i, s := range stations {
			fav := "☆"
			if s.Favorite {
				fav = CurrentTheme.Highlight.Sprint("★")
			}
			tbl.AddRow(i+1, s.ID, s.Name, s.Country, s.Genre, fav)
		}
		tbl.Print()
		fmt.Fprintln(Output)
	}

	fmt.Fprintf(Output, "  %s\n", services.L.Get("total_stations", map[string]interface{}{"count": len(stations)}))
	if subtitle != "" {
		PrintInfo(subtitle)
	}
}

// runesTruncate truncates s to at most n runes (not bytes).
// Handles Turkish / Latin extended characters correctly.
func runesTruncate(s string, n int) string {
	rr := []rune(s)
	if len(rr) <= n {
		return s
	}
	return string(rr[:n-1]) + "…"
}

// firstNonEmptyStr returns the first non-empty string from vals.
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func PrintNowPlaying(station *models.RadioStation, song string, volume int, isMuted bool, isRecording bool) {
	volText := fmt.Sprintf("%%%d", volume)
	if isMuted {
		volText += fmt.Sprintf(" (%s)", services.L.Get("muted"))
	}

	fmt.Fprintln(Output)
	CurrentTheme.Primary.Fprintf(Output, "  [ %s ]\n", strings.ToUpper(services.L.Get("now_playing")))
	fmt.Fprintf(Output, "  %-10s: %s\n", services.L.Get("station"), station.Name)
	fmt.Fprintf(Output, "  %-10s: %s\n", services.L.Get("country"), station.Country)
	fmt.Fprintf(Output, "  %-10s: %s\n", services.L.Get("genre"), station.Genre)
	if song != "" {
		CurrentTheme.Highlight.Fprintf(Output, "  %-10s: %s\n", services.L.Get("song"), song)
	}
	fmt.Fprintf(Output, "  %-10s: %s", services.L.Get("volume"), volText)
	if isRecording {
		fmt.Fprint(Output, " | ")
		CurrentTheme.Error.Fprintf(Output, "● %s", services.L.Get("recording"))
	}
	fmt.Fprintln(Output)
}
