package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/tui"
	"radio-shell/internal/ui"
	"radio-shell/internal/web"
)

const webURL = "http://127.0.0.1:8765"

type webProcessInfo struct {
	PID int    `json:"pid"`
	URL string `json:"url"`
}

func Run() {
	webMode := flag.Bool("web", false, "Start the web interface")
	foreground := flag.Bool("foreground", false, "Run web server in foreground")
	killWeb := flag.Bool("kill", false, "Stop the background web server")
	flag.Parse()

	cfg := config.NewRadioConfig()
	if err := cfg.EnsureDirs(); err != nil {
		fmt.Printf("Error creating directories: %v\n", err)
		os.Exit(1)
	}

	if *killWeb {
		stopBackgroundWebServer(cfg)
		return
	}

	initialLang := ensureLanguage(cfg)

	settingsSvc := services.NewSettingsService(cfg)
	if initialLang != "" {
		settingsSvc.SetLanguage(initialLang)
	}
	services.L.SetLanguage(settingsSvc.GetLanguage())
	ui.LoadTheme()

	stationSvc := services.NewStationService(cfg)
	if err := stationSvc.Init(); err != nil {
		ui.PrintError(fmt.Sprintf("İstasyonlar yüklenemedi: %v", err))
	}

	statsSvc := services.NewStatisticsService(cfg)
	rbSvc := services.NewRadioBrowserService()
	nsSvc := services.NewNotificationService(settingsSvc)
	audioPlayer := player.NewAudioPlayer(cfg, nsSvc)
	sysSvc := services.NewSystemService()

	if *webMode {
		if !*foreground {
			startBackgroundWebServer(cfg)
			return
		}

		writeWebProcessInfo(cfg, os.Getpid())
		defer removeWebProcessInfo(cfg, os.Getpid())

		server := web.NewWebServer(audioPlayer, stationSvc, settingsSvc, sysSvc)
		fmt.Printf("Starting web server on %s\n", webURL)
		if os.Getenv("RADIO_WEB_NO_BROWSER") != "1" {
			time.AfterFunc(1500*time.Millisecond, func() { openBrowser(webURL) })
		}
		if err := server.Start("127.0.0.1", 8765); err != nil {
			fmt.Printf("Web server error: %v\n", err)
		}
		return
	}

	if err := tui.Run(stationSvc, statsSvc, sysSvc, settingsSvc, rbSvc, nsSvc, audioPlayer); err != nil {
		ui.PrintError(fmt.Sprintf("TUI hatası: %v", err))
	}
	audioPlayer.Stop()
}

func startBackgroundWebServer(cfg *config.RadioConfig) {
	if pid, ok := getRunningWebPID(cfg); ok {
		ui.PrintSuccess(fmt.Sprintf("Web sunucusu zaten çalışıyor (PID: %d)", pid))
		ui.PrintInfo(fmt.Sprintf("Tarayıcı açılıyor: %s", webURL))
		openBrowser(webURL)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Çalıştırılabilir dosya bulunamadı: %v", err))
		return
	}

	cmd := exec.Command(exe, "--web", "--foreground")
	cmd.Env = append(os.Environ(), "RADIO_WEB_NO_BROWSER=1")
	if err := cmd.Start(); err != nil {
		ui.PrintError(fmt.Sprintf("Web sunucusu başlatılamadı: %v", err))
		return
	}

	writeWebProcessInfo(cfg, cmd.Process.Pid)
	ui.PrintSuccess(fmt.Sprintf("Web sunucusu arka planda çalışıyor (PID: %d)", cmd.Process.Pid))
	ui.PrintInfo(fmt.Sprintf("Tarayıcı açılıyor: %s", webURL))
	time.Sleep(1500 * time.Millisecond)
	openBrowser(webURL)
}

func ensureLanguage(cfg *config.RadioConfig) string {
	if languageAlreadyConfigured(cfg) || !isTerminalInput() {
		return ""
	}

	ui.PrintHeader("LANGUAGE SELECTION / DİL SEÇİMİ")
	fmt.Println()
	fmt.Println("  Please select your language / Lütfen dilinizi seçin:")
	fmt.Println()

	languages := services.L.GetLanguages()
	codes := make([]string, 0, len(languages))
	for code := range languages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		fmt.Printf("  %-4s - %s\n", code, languages[code])
	}

	fmt.Print("\n  Selection [en]: ")
	var choice string
	if _, err := fmt.Scanln(&choice); err != nil {
		return "en"
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return "en"
	}
	if _, ok := languages[choice]; ok {
		return choice
	}
	return "en"
}

func languageAlreadyConfigured(cfg *config.RadioConfig) bool {
	data, err := os.ReadFile(cfg.SettingsFile)
	if err != nil {
		return false
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}
	lang, ok := settings["language"].(string)
	return ok && strings.TrimSpace(lang) != ""
}

func isTerminalInput() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func stopBackgroundWebServer(cfg *config.RadioConfig) bool {
	info, ok := readWebProcessInfo(cfg)
	if !ok {
		ui.PrintInfo("Çalışan web sunucusu bulunamadı.")
		return false
	}

	process, err := os.FindProcess(info.PID)
	if err != nil || !processExists(process) {
		removeWebProcessInfo(cfg, info.PID)
		ui.PrintInfo("Eski web PID dosyası temizlendi; çalışan web sunucusu yok.")
		return false
	}

	ui.PrintInfo(fmt.Sprintf("Web sunucusu durduruluyor (PID: %d)...", info.PID))
	_ = process.Signal(os.Interrupt)
	for i := 0; i < 50; i++ {
		if !processExists(process) {
			removeWebProcessInfo(cfg, info.PID)
			ui.PrintSuccess("Web sunucusu durduruldu.")
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = process.Kill()
	removeWebProcessInfo(cfg, info.PID)
	ui.PrintSuccess("Web sunucusu durduruldu.")
	return true
}

func getRunningWebPID(cfg *config.RadioConfig) (int, bool) {
	info, ok := readWebProcessInfo(cfg)
	if !ok {
		return 0, false
	}

	process, err := os.FindProcess(info.PID)
	if err != nil || !processExists(process) {
		removeWebProcessInfo(cfg, info.PID)
		return 0, false
	}
	return info.PID, true
}

func readWebProcessInfo(cfg *config.RadioConfig) (webProcessInfo, bool) {
	data, err := os.ReadFile(cfg.WebPIDFile)
	if err != nil {
		return webProcessInfo{}, false
	}

	var info webProcessInfo
	if err := json.Unmarshal(data, &info); err == nil && info.PID > 0 {
		return info, true
	}

	var legacyPID int
	if _, err := fmt.Sscanf(string(data), "%d", &legacyPID); err == nil && legacyPID > 0 {
		return webProcessInfo{PID: legacyPID, URL: webURL}, true
	}
	return webProcessInfo{}, false
}

func writeWebProcessInfo(cfg *config.RadioConfig, pid int) {
	_ = os.MkdirAll(cfg.AppDir, 0755)
	data, err := json.MarshalIndent(webProcessInfo{PID: pid, URL: webURL}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cfg.WebPIDFile, data, 0644)
}

func removeWebProcessInfo(cfg *config.RadioConfig, expectedPID int) {
	info, ok := readWebProcessInfo(cfg)
	if ok && expectedPID > 0 && info.PID != expectedPID {
		return
	}
	_ = os.Remove(cfg.WebPIDFile)
}

func processExists(process *os.Process) bool {
	if process == nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return process.Signal(syscall.Signal(0)) == nil
	}
	err := process.Signal(syscall.Signal(0))
	return err == nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
