package main

import (
	"flag"
	"fmt"
	"os"
	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/shell"
	"radio-shell/internal/ui"
	"radio-shell/internal/web"
)

func main() {
	webMode := flag.Bool("web", false, "Start the web interface")
	flag.Parse()

	// 1. Initialize Configuration
	cfg := config.NewRadioConfig()
	if err := cfg.EnsureDirs(); err != nil {
		fmt.Printf("Error creating directories: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Services
	settingsSvc := services.NewSettingsService(cfg)
	services.L.SetLanguage(settingsSvc.GetLanguage())
	ui.LoadTheme()

	stationSvc := services.NewStationService(cfg)
	stationSvc.Init()

	statsSvc := services.NewStatisticsService(cfg)
	rbSvc := services.NewRadioBrowserService()
	nsSvc := services.NewNotificationService(settingsSvc)

	// 3. Initialize Player
	audioPlayer := player.NewAudioPlayer(cfg, nsSvc)
	sysSvc := services.NewSystemService()

	if *webMode {
		server := web.NewWebServer(audioPlayer, stationSvc, settingsSvc, sysSvc)
		fmt.Println("Starting web server on http://127.0.0.1:8765")
		if err := server.Start("127.0.0.1", 8765); err != nil {
			fmt.Printf("Web server error: %v\n", err)
		}
	} else {
		// 4. Initialize and run Shell
		sh := shell.NewInteractiveShell(stationSvc, audioPlayer)
		shell.RegisterAllCommands(sh, stationSvc, statsSvc, sysSvc, settingsSvc, rbSvc, nsSvc, audioPlayer)

		sh.Run()
		audioPlayer.Stop()
	}
}
