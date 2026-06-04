package ui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"radio-shell/internal/models"
	"radio-shell/internal/services"
	"strings"

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
	"winamp-classic": {
		Primary:   color.New(color.FgGreen),
		Secondary: color.New(color.FgBlue),
		Highlight: color.New(color.FgYellow),
		Success:   color.New(color.FgHiGreen),
		Error:     color.New(color.FgHiRed),
	},
}

var (
	CurrentTheme     = Themes["default"]
	CurrentThemeName = "default"
)

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

func PrintBanner() {
	width := 66
	appTitle := services.L.Get("app_title")

	fmt.Println()
	CurrentTheme.Secondary.Printf("  ╔%s╗\n", strings.Repeat("═", width))
	CurrentTheme.Secondary.Print("  ║")
	fmt.Print("                  ♬  ")
	CurrentTheme.Highlight.Print("░░░ RADIO SHELL ░░░")
	fmt.Print("  ♬                   ")
	CurrentTheme.Secondary.Println("║")

	CurrentTheme.Secondary.Print("  ║")
	padding := (width - len(appTitle)) / 2
	fmt.Print(strings.Repeat(" ", padding))
	CurrentTheme.Primary.Print(appTitle)
	fmt.Print(strings.Repeat(" ", width-padding-len(appTitle)))
	CurrentTheme.Secondary.Println("║")

	CurrentTheme.Secondary.Print("  ║")
	version := "v3.0.0 | Go 1.25 + Fatih Color"
	paddingVer := (width - len(version)) / 2
	fmt.Print(strings.Repeat(" ", paddingVer))
	fmt.Print(version)
	fmt.Print(strings.Repeat(" ", width-paddingVer-len(version)))
	CurrentTheme.Secondary.Println("║")

	CurrentTheme.Secondary.Printf("  ╚%s╝\n", strings.Repeat("═", width))
	fmt.Println()
}

func PrintHeader(title string) {
	fmt.Println()
	CurrentTheme.Highlight.Print(" ❯❯ ")
	CurrentTheme.Primary.Print(strings.ToUpper(title))
	CurrentTheme.Highlight.Println(" ❮❮ ")
	fmt.Println(strings.Repeat("─", 40))
}

func PrintError(msg string) {
	CurrentTheme.Error.Printf("  ✘ %s\n", msg)
}

func PrintSuccess(msg string) {
	CurrentTheme.Success.Printf("  ✔ %s\n", msg)
}

func PrintInfo(msg string) {
	CurrentTheme.Highlight.Printf("  ℹ %s\n", msg)
}

func PrintStationTable(title string, stations []models.RadioStation, subtitle string) {
	if len(stations) == 0 {
		PrintInfo(services.L.Get("no_stations"))
		return
	}

	PrintHeader(title)

	tbl := table.New("NO", "ID", "STATION NAME", "COUNTRY", "GENRE", "FAV")
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
	fmt.Printf("\n  %s\n", services.L.Get("total_stations", map[string]interface{}{"count": len(stations)}))
	if subtitle != "" {
		PrintInfo(subtitle)
	}
	fmt.Println()
}

func PrintNowPlaying(station *models.RadioStation, song string, volume int, isMuted bool, isRecording bool) {
	volText := fmt.Sprintf("%%%d", volume)
	if isMuted {
		volText += fmt.Sprintf(" (%s)", services.L.Get("muted"))
	}

	fmt.Println()
	CurrentTheme.Primary.Printf("  [ %s ]\n", strings.ToUpper(services.L.Get("now_playing")))
	fmt.Printf("  %-10s: %s\n", services.L.Get("station"), station.Name)
	fmt.Printf("  %-10s: %s\n", services.L.Get("country"), station.Country)
	fmt.Printf("  %-10s: %s\n", services.L.Get("genre"), station.Genre)
	if song != "" {
		CurrentTheme.Highlight.Printf("  %-10s: %s\n", services.L.Get("song"), song)
	}
	fmt.Printf("  %-10s: %s", services.L.Get("volume"), volText)
	if isRecording {
		fmt.Print(" | ")
		CurrentTheme.Error.Printf("● %s", services.L.Get("recording"))
	}
	fmt.Println()
}
