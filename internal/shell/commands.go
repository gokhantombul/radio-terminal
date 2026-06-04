package shell

import (
	"flag"
	"fmt"
	"radio-shell/internal/models"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/ui"
	"strconv"
	"strings"
)

type Commands struct {
	shell           *InteractiveShell
	stationService  *services.StationService
	statsService    *services.StatisticsService
	systemService   *services.SystemService
	settingsService *services.SettingsService
	radioBrowser    *services.RadioBrowserService
	notificationSvc *services.NotificationService
	player          *player.AudioPlayer
}

func RegisterAllCommands(sh *InteractiveShell, ss *services.StationService, stats *services.StatisticsService, sys *services.SystemService, set *services.SettingsService, rb *services.RadioBrowserService, ns *services.NotificationService, p *player.AudioPlayer) {
	c := &Commands{
		shell:           sh,
		stationService:  ss,
		statsService:    stats,
		systemService:   sys,
		settingsService: set,
		radioBrowser:    rb,
		notificationSvc: ns,
		player:          p,
	}

	// Basic
	sh.Register("listele", c.Listele, "cmd_listele_desc", "cat_listing")
	sh.Register("turkiye", c.Turkiye, "cmd_turkiye_desc", "cat_listing")
	sh.Register("ulkeler", c.Ulkeler, "cmd_ulkeler_desc", "cat_listing")
	sh.Register("ulke", c.Ulke, "cmd_ulke_desc", "cat_listing")
	sh.Register("turler", c.Turler, "cmd_turler_desc", "cat_listing")
	sh.Register("tur", c.Tur, "cmd_tur_desc", "cat_listing")
	sh.Register("ara", c.Ara, "cmd_ara_desc", "cat_listing")
	sh.Register("istatistik", c.Istatistik, "cmd_istatistik_desc", "cat_management")
	sh.Register("durum", c.Durum, "cmd_durum_desc", "cat_playback")
	sh.Register("sistem", c.Sistem, "cmd_sistem_desc", "cat_management")
	sh.Register("clear", c.Clear, "cmd_temizle_desc", "cat_management")
	sh.Register("temizle", c.Clear, "cmd_temizle_desc", "cat_management")

	// Playback
	sh.Register("cal", c.Cal, "cmd_cal_desc", "cat_playback")
	sh.Register("dur", c.Dur, "cmd_durdur_desc", "cat_playback")
	sh.Register("durdur", c.Dur, "cmd_durdur_desc", "cat_playback")
	sh.Register("ses", c.Ses, "cmd_ses_desc", "cat_playback")
	sh.Register("sessiz", c.Sessiz, "cmd_sessiz_desc", "cat_playback")

	// Recording
	sh.Register("kaydet", c.Kaydet, "cmd_kaydet_desc", "cat_recording")
	sh.Register("kayitdur", c.Kayitdur, "cmd_kayitdur_desc", "cat_recording")

	// Management
	sh.Register("favori", c.Favori, "cmd_favori_desc", "cat_management")
	sh.Register("favoriler", c.Favoriler, "cmd_favoriler_desc", "cat_management")
	sh.Register("tema", c.Tema, "cmd_tema_desc", "cat_management")
	sh.Register("ekle", c.Ekle, "cmd_ekle_desc", "cat_management")
	sh.Register("sil", c.Sil, "cmd_sil_desc", "cat_management")
	sh.Register("online-ara", c.OnlineAra, "cmd_online_ara_desc", "cat_management")
	sh.Register("online-ekle", c.OnlineEkle, "cmd_online_ekle_desc", "cat_management")
}

func (c *Commands) Listele(args []string) {
	fs := flag.NewFlagSet("listele", flag.ContinueOnError)
	n := fs.Int("n", 50, "number of stations")
	hepsi := fs.Bool("hepsi", false, "all stations")
	if err := fs.Parse(args); err != nil {
		return
	}

	stations := c.stationService.GetAllStations()
	total := len(stations)

	var shown []models.RadioStation
	subtitle := ""
	if *hepsi {
		shown = stations
	} else {
		limit := *n
		if limit > total {
			limit = total
		}
		shown = stations[:limit]
		if total > limit {
			subtitle = services.L.Get("list_subtitle", map[string]interface{}{"limit": limit, "total": total})
		}
	}

	c.shell.UpdateLastList(shown)
	ui.PrintStationTable(services.L.Get("all_stations"), shown, subtitle)
}

func (c *Commands) Turkiye(args []string) {
	all := c.stationService.GetAllStations()
	var stations []models.RadioStation
	for _, s := range all {
		if strings.ToLower(s.Country) == "türkiye" || strings.ToLower(s.Country) == "turkey" {
			stations = append(stations, s)
		}
	}
	c.shell.UpdateLastList(stations)
	ui.PrintStationTable(services.L.Get("tr_stations"), stations, "")
}

func (c *Commands) Ulkeler(args []string) {
	countries := c.stationService.GetCountries()
	ui.CurrentTheme.Primary.Printf("%s:\n", services.L.Get("countries_list"))
	for _, country := range countries {
		fmt.Printf("  - %s\n", country)
	}
}

func (c *Commands) Ulke(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: ulke <country>")
		return
	}
	name := strings.Join(args, " ")
	stations := c.stationService.Search(name)
	var filtered []models.RadioStation
	for _, s := range stations {
		if strings.ToLower(s.Country) == strings.ToLower(name) {
			filtered = append(filtered, s)
		}
	}
	c.shell.UpdateLastList(filtered)
	ui.PrintStationTable(fmt.Sprintf("%s: %s", services.L.Get("country"), name), filtered, "")
}

func (c *Commands) Turler(args []string) {
	genres := c.stationService.GetGenres()
	ui.CurrentTheme.Primary.Printf("%s:\n", services.L.Get("genres_list"))
	for _, genre := range genres {
		fmt.Printf("  - %s\n", genre)
	}
}

func (c *Commands) Tur(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: tur <genre>")
		return
	}
	name := strings.Join(args, " ")
	stations := c.stationService.Search(name)
	c.shell.UpdateLastList(stations)
	ui.PrintStationTable(fmt.Sprintf("%s: %s", services.L.Get("genre"), name), stations, "")
}

func (c *Commands) Ara(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: ara <query>")
		return
	}
	query := strings.Join(args, " ")
	stations := c.stationService.Search(query)
	c.shell.UpdateLastList(stations)
	ui.PrintStationTable(fmt.Sprintf("%s > %s", services.L.Get("cat_listing"), query), stations, "")
}

func (c *Commands) Istatistik(args []string) {
	top := c.statsService.GetTopStations(10)
	totalTime := c.statsService.GetTotalListenTime()
	sessions := c.statsService.GetTotalSessions()

	ui.CurrentTheme.Primary.Printf("%s: ", services.L.Get("stats_total_time"))
	fmt.Println(totalTime)
	ui.CurrentTheme.Primary.Printf("%s: ", services.L.Get("stats_total_sessions"))
	fmt.Println(sessions)

	if len(top) == 0 {
		ui.PrintInfo(services.L.Get("stats_no_data"))
		return
	}

	ui.PrintHeader(services.L.Get("stats_top_title"))
	for _, s := range top {
		fmt.Printf("  %-30s | %d s | %d sessions\n", s.StationName, s.TotalSeconds, s.SessionCount)
	}
}

func (c *Commands) Durum(args []string) {
	station, song, vol, muted, playing, recording, _ := c.player.GetStatus()
	if !playing || station == nil {
		ui.PrintInfo(services.L.Get("msg_no_playing_station"))
		return
	}
	ui.PrintNowPlaying(station, song, vol, muted, recording)
}

func (c *Commands) Sistem(args []string) {
	mem := c.systemService.GetMemoryInfo()
	stats := c.systemService.GetSystemStats()

	ui.PrintHeader(services.L.Get("sys_info_title"))
	fmt.Printf("  %-20s: %s\n", services.L.Get("sys_os"), stats["os"])
	fmt.Printf("  %-20s: %v\n", "Go Version", stats["go_version"])
	fmt.Printf("  %-20s: %.2f%%\n", services.L.Get("sys_cpu"), stats["cpu_percent"])
	fmt.Printf("  %-20s: %s\n", services.L.Get("sys_total_mem"), c.systemService.FormatBytes(mem["total_memory"].(uint64)))
}

func (c *Commands) Clear(args []string) {
	fmt.Print("\033[H\033[2J")
	ui.PrintBanner()
}

func (c *Commands) Cal(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: cal <id_or_index>")
		return
	}

	idOrIdx := args[0]
	var st *models.RadioStation

	// Try as index
	if idx, err := strconv.Atoi(idOrIdx); err == nil {
		lastList := c.shell.GetLastList()
		if idx > 0 && idx <= len(lastList) {
			st = &lastList[idx-1]
		}
	}

	// Try as ID if not found by index
	if st == nil {
		st = c.stationService.GetStation(idOrIdx)
	}

	if st == nil {
		ui.PrintError(services.L.Get("msg_station_not_found"))
		return
	}

	ui.PrintSuccess(services.L.Get("msg_playing", map[string]interface{}{"name": st.Name}))
	c.player.Play(*st, c.settingsService.GetVolume(), c.settingsService.IsMuted())
	c.settingsService.SetLastStationID(st.ID)
}

func (c *Commands) Dur(args []string) {
	c.player.Stop()
	ui.PrintInfo(services.L.Get("msg_stop_playing"))
}

func (c *Commands) Ses(args []string) {
	if len(args) == 0 {
		ui.PrintInfo(fmt.Sprintf("%s: %%%d", services.L.Get("volume"), c.settingsService.GetVolume()))
		return
	}

	vol, err := strconv.Atoi(args[0])
	if err != nil || vol < 0 || vol > 100 {
		ui.PrintError("Volume must be 0-100")
		return
	}

	c.player.SetVolume(vol, true)
	c.settingsService.SetVolume(vol)
	c.settingsService.SetMuted(false)
	ui.PrintSuccess(services.L.Get("msg_vol_set", map[string]interface{}{"vol": vol}))
}

func (c *Commands) Sessiz(args []string) {
	_, _, _, muted, _, _, _ := c.player.GetStatus()
	newMuted := !muted
	c.player.SetMuted(newMuted)
	c.settingsService.SetMuted(newMuted)

	if newMuted {
		ui.PrintInfo(services.L.Get("msg_muted"))
	} else {
		ui.PrintInfo(services.L.Get("msg_unmuted"))
	}
}

func (c *Commands) Favori(args []string) {
	id := ""
	if len(args) > 0 {
		id = args[0]
	} else {
		st, _, _, _, _, _, _ := c.player.GetStatus()
		if st != nil {
			id = st.ID
		}
	}

	if id == "" {
		ui.PrintError(services.L.Get("msg_station_not_found"))
		return
	}

	added := c.stationService.ToggleFavorite(id)
	st := c.stationService.GetStation(id)
	if added {
		ui.PrintSuccess(services.L.Get("msg_fav_added", map[string]interface{}{"name": st.Name}))
	} else {
		ui.PrintInfo(services.L.Get("msg_fav_removed", map[string]interface{}{"name": st.Name}))
	}
}

func (c *Commands) Favoriler(args []string) {
	favs := c.stationService.GetFavorites()
	c.shell.UpdateLastList(favs)
	ui.PrintStationTable(services.L.Get("favoriler"), favs, "")
}

func (c *Commands) Tema(args []string) {
	if len(args) == 0 {
		ui.PrintInfo("Available themes: default, hacker, winamp-classic")
		return
	}

	if ui.SetTheme(args[0]) {
		ui.PrintSuccess("Theme updated.")
	} else {
		ui.PrintError("Theme not found.")
	}
}

func (c *Commands) Kaydet(args []string) {
	file, err := c.player.StartRecording()
	if err != nil {
		ui.PrintError(services.L.Get("msg_recording_failed", map[string]interface{}{"error": err}))
		return
	}
	ui.PrintSuccess(services.L.Get("msg_recording_started", map[string]interface{}{"file": file}))
}

func (c *Commands) Kayitdur(args []string) {
	path := c.player.StopRecording()
	if path == "" {
		ui.PrintInfo(services.L.Get("msg_no_active_record"))
		return
	}
	ui.PrintSuccess(services.L.Get("msg_recording_stopped", map[string]interface{}{"path": path}))
}

func (c *Commands) Ekle(args []string) {
	fs := flag.NewFlagSet("ekle", flag.ContinueOnError)
	id := fs.String("id", "", "station id")
	name := fs.String("isim", "", "station name")
	country := fs.String("ulke", "", "country")
	genre := fs.String("tur", "", "genre")
	url := fs.String("url", "", "stream url")

	if err := fs.Parse(args); err != nil {
		return
	}

	if *id == "" || *name == "" || *url == "" {
		ui.PrintError("Usage: ekle --id ID --isim NAME --url URL [--ulke COUNTRY] [--tur GENRE]")
		return
	}

	st := models.RadioStation{
		ID:      *id,
		Name:    *name,
		Country: *country,
		Genre:   *genre,
		URL:     *url,
	}
	c.stationService.AddCustomStation(st)
	ui.PrintSuccess(services.L.Get("msg_station_added", map[string]interface{}{"name": st.Name}))
}

func (c *Commands) Sil(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: sil <id>")
		return
	}

	id := args[0]
	if c.stationService.RemoveCustomStation(id) {
		ui.PrintSuccess(services.L.Get("msg_station_deleted", map[string]interface{}{"id": id}))
	} else {
		ui.PrintError(services.L.Get("msg_station_not_found"))
	}
}

var lastOnlineResults []services.OnlineStation

func (c *Commands) OnlineAra(args []string) {
	if len(args) == 0 {
		ui.PrintError("Usage: online-ara <query>")
		return
	}

	query := strings.Join(args, " ")
	ui.PrintInfo(services.L.Get("msg_searching"))
	results, err := c.radioBrowser.Search(query, "", "", 20)
	if err != nil {
		ui.PrintError(err.Error())
		return
	}

	lastOnlineResults = results
	ui.PrintHeader(services.L.Get("msg_search_results"))
	for i, s := range results {
		fmt.Printf("  %2d. [%s] %s (%s)\n", i+1, s.CountryDisplay(), s.Name, s.GenreDisplay())
	}
	ui.PrintInfo(services.L.Get("msg_online_add_hint"))
}

func (c *Commands) OnlineEkle(args []string) {
	fs := flag.NewFlagSet("online-ekle", flag.ContinueOnError)
	n := fs.Int("n", 0, "index from search results")
	if err := fs.Parse(args); err != nil {
		return
	}

	if *n <= 0 || *n > len(lastOnlineResults) {
		ui.PrintError("Invalid index.")
		return
	}

	os := lastOnlineResults[*n-1]
	st := models.RadioStation{
		ID:      os.UUID,
		Name:    os.Name,
		Country: os.Country,
		Genre:   os.GenreDisplay(),
		URL:     os.URL,
	}
	c.stationService.AddCustomStation(st)
	ui.PrintSuccess(services.L.Get("msg_station_added", map[string]interface{}{"name": st.Name}))
}
