package shell

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os/exec"
	"radio-shell/internal/models"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"radio-shell/internal/ui"
	"radio-shell/internal/web"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Commands struct {
	shell           CommandHost
	stationService  *services.StationService
	statsService    *services.StatisticsService
	systemService   *services.SystemService
	settingsService *services.SettingsService
	radioBrowser    *services.RadioBrowserService
	notificationSvc *services.NotificationService
	player          *player.AudioPlayer

	lastOnlineResults []services.OnlineStation

	sessionMu      sync.Mutex
	sessionStart   time.Time
	sessionStation *models.RadioStation

	songMu      sync.Mutex
	songHistory []string

	sleepMu    sync.Mutex
	sleepTimer *time.Timer

	webMu      sync.Mutex
	webStarted bool

	rng *rand.Rand
}

type CommandHost interface {
	Register(name string, f CommandFunc, desc, category string)
	SetOnExit(func())
	UpdateLastList([]models.RadioStation)
	GetLastList() []models.RadioStation
}

func RegisterAllCommands(sh CommandHost, ss *services.StationService, stats *services.StatisticsService, sys *services.SystemService, set *services.SettingsService, rb *services.RadioBrowserService, ns *services.NotificationService, p *player.AudioPlayer) {
	c := &Commands{
		shell:           sh,
		stationService:  ss,
		statsService:    stats,
		systemService:   sys,
		settingsService: set,
		radioBrowser:    rb,
		notificationSvc: ns,
		player:          p,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
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
	sh.Register("web", c.Web, "cmd_web_desc", "cat_management")

	// Playback
	sh.Register("cal", c.Cal, "cmd_cal_desc", "cat_playback")
	sh.Register("son", c.Son, "cmd_son_desc", "cat_playback")
	sh.Register("dur", c.Dur, "cmd_durdur_desc", "cat_playback")
	sh.Register("durdur", c.Dur, "cmd_durdur_desc", "cat_playback")
	sh.Register("ses", c.Ses, "cmd_ses_desc", "cat_playback")
	sh.Register("sessiz", c.Sessiz, "cmd_sessiz_desc", "cat_playback")
	sh.Register("mute", c.Sessiz, "cmd_sessiz_desc", "cat_playback")
	sh.Register("sonraki", c.Sonraki, "cmd_sonraki_desc", "cat_playback")
	sh.Register("ileri", c.Sonraki, "cmd_sonraki_desc", "cat_playback")
	sh.Register("onceki", c.Onceki, "cmd_onceki_desc", "cat_playback")
	sh.Register("geri", c.Onceki, "cmd_onceki_desc", "cat_playback")
	sh.Register("karistir", c.Karistir, "cmd_karistir_desc", "cat_playback")
	sh.Register("rastgele", c.Karistir, "cmd_karistir_desc", "cat_playback")
	sh.Register("uyku", c.Uyku, "cmd_uyku_desc", "cat_playback")
	sh.Register("gecmis", c.Gecmis, "cmd_gecmis_desc", "cat_playback")

	// Recording
	sh.Register("kaydet", c.Kaydet, "cmd_kaydet_desc", "cat_recording")
	sh.Register("kayitdur", c.Kayitdur, "cmd_kayitdur_desc", "cat_recording")

	// Management
	sh.Register("favori", c.Favori, "cmd_favori_desc", "cat_management")
	sh.Register("favoriler", c.Favoriler, "cmd_favoriler_desc", "cat_management")
	sh.Register("tema", c.Tema, "cmd_tema_desc", "cat_management")
	sh.Register("kontrol", c.Kontrol, "cmd_kontrol_desc", "cat_management")
	sh.Register("ekle", c.Ekle, "cmd_ekle_desc", "cat_management")
	sh.Register("duzenle", c.Duzenle, "cmd_duzenle_desc", "cat_management")
	sh.Register("sil", c.Sil, "cmd_sil_desc", "cat_management")
	sh.Register("iceaktar", c.Iceaktar, "cmd_iceaktar_desc", "cat_management")
	sh.Register("bildirim", c.Bildirim, "cmd_bildirim_desc", "cat_management")
	sh.Register("online-ara", c.OnlineAra, "cmd_online_ara_desc", "cat_listing")
	sh.Register("online-ekle", c.OnlineEkle, "cmd_online_ekle_desc", "cat_management")
	sh.Register("dil", c.Dil, "cmd_dil_desc", "cat_management")
	sh.Register("lang", c.Dil, "cmd_dil_desc", "cat_management")

	p.SetOnSongChange(c.recordSong)
	sh.SetOnExit(c.recordSession)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c *Commands) playStation(st models.RadioStation) {
	c.recordSession()
	ui.PrintInfo(fmt.Sprintf("%s %s", services.L.Get("connecting"), st.Name))
	c.player.Play(st, c.settingsService.GetVolume(), c.settingsService.IsMuted())
	c.settingsService.SetLastStationID(st.ID)

	stationCopy := st
	c.sessionMu.Lock()
	c.sessionStation = &stationCopy
	c.sessionStart = time.Now()
	c.sessionMu.Unlock()

	ui.PrintSuccess(services.L.Get("msg_playing", map[string]interface{}{"name": st.Name}))
}

func (c *Commands) recordSession() {
	c.sessionMu.Lock()
	station := c.sessionStation
	start := c.sessionStart
	c.sessionStation = nil
	c.sessionStart = time.Time{}
	c.sessionMu.Unlock()

	if station != nil && !start.IsZero() {
		c.statsService.RecordSession(*station, time.Since(start))
	}
}

func (c *Commands) recordSong(title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}

	c.songMu.Lock()
	defer c.songMu.Unlock()

	if len(c.songHistory) > 0 && c.songHistory[len(c.songHistory)-1] == title {
		return
	}
	c.songHistory = append(c.songHistory, title)
	if len(c.songHistory) > 50 {
		c.songHistory = c.songHistory[len(c.songHistory)-50:]
	}
}

func (c *Commands) currentStation() *models.RadioStation {
	station, _, _, _, _, _, _ := c.player.GetStatus()
	return station
}

func (c *Commands) openBrowser(url string) {
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

func (c *Commands) stationActive(st models.RadioStation) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodHead, st.URL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "VLC/3.0.16 LibVLC/3.0.16")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	active := isActiveHTTPStatus(resp.StatusCode)
	shouldRetryGet := resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusForbidden
	resp.Body.Close()
	if active || !shouldRetryGet {
		return active
	}

	req, err = http.NewRequest(http.MethodGet, st.URL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "VLC/3.0.16 LibVLC/3.0.16")
	req.Header.Set("Range", "bytes=0-0")

	resp, err = client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return isActiveHTTPStatus(resp.StatusCode)
}

func isActiveHTTPStatus(status int) bool {
	return status == http.StatusOK ||
		status == http.StatusFound ||
		status == http.StatusMovedPermanently ||
		status == http.StatusTemporaryRedirect ||
		status == http.StatusPermanentRedirect ||
		status == http.StatusPartialContent
}

func (c *Commands) Listele(args []string) {
	fs := newFlagSet("listele")
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
	if len(countries) == 0 {
		ui.PrintInfo(services.L.Get("msg_no_match"))
		return
	}
	ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s:\n", services.L.Get("countries_list"))
	all := c.stationService.GetAllStations()
	for _, country := range countries {
		count := 0
		for _, st := range all {
			if st.Country == country {
				count++
			}
		}
		ui.Fprintf("  - %s (%d %s)\n", country, count, strings.ToLower(services.L.Get("station")))
	}
}

func (c *Commands) Ulke(args []string) {
	fs := newFlagSet("ulke")
	nameFlag := fs.String("i", "", "country")
	nameLong := fs.String("isim", "", "country")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: ulke -i <country>")
		return
	}

	name := firstNonEmpty(*nameFlag, *nameLong, strings.Join(fs.Args(), " "))
	if name == "" {
		ui.PrintError("Usage: ulke -i <country>")
		return
	}
	var filtered []models.RadioStation
	for _, s := range c.stationService.GetAllStations() {
		if strings.ToLower(s.Country) == strings.ToLower(name) {
			filtered = append(filtered, s)
		}
	}
	c.shell.UpdateLastList(filtered)
	ui.PrintStationTable(fmt.Sprintf("%s: %s", services.L.Get("country"), name), filtered, "")
}

func (c *Commands) Turler(args []string) {
	genres := c.stationService.GetGenres()
	if len(genres) == 0 {
		ui.PrintInfo(services.L.Get("msg_no_match"))
		return
	}
	ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s:\n", services.L.Get("genres_list"))
	all := c.stationService.GetAllStations()
	for _, genre := range genres {
		count := 0
		for _, st := range all {
			if strings.Contains(strings.ToLower(st.Genre), strings.ToLower(genre)) {
				count++
			}
		}
		ui.Fprintf("  - %s (%d %s)\n", genre, count, strings.ToLower(services.L.Get("station")))
	}
}

func (c *Commands) Tur(args []string) {
	fs := newFlagSet("tur")
	nameFlag := fs.String("i", "", "genre")
	nameLong := fs.String("isim", "", "genre")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: tur -i <genre>")
		return
	}

	name := firstNonEmpty(*nameFlag, *nameLong, strings.Join(fs.Args(), " "))
	if name == "" {
		ui.PrintError("Usage: tur -i <genre>")
		return
	}
	var stations []models.RadioStation
	for _, s := range c.stationService.GetAllStations() {
		if strings.Contains(strings.ToLower(s.Genre), strings.ToLower(name)) {
			stations = append(stations, s)
		}
	}
	c.shell.UpdateLastList(stations)
	ui.PrintStationTable(fmt.Sprintf("%s: %s", services.L.Get("genre"), name), stations, "")
}

func (c *Commands) Ara(args []string) {
	fs := newFlagSet("ara")
	queryShort := fs.String("s", "", "query")
	queryLong := fs.String("sorgu", "", "query")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: ara -s <query>")
		return
	}

	query := firstNonEmpty(*queryShort, *queryLong, strings.Join(fs.Args(), " "))
	if query == "" {
		ui.PrintError("Usage: ara -s <query>")
		return
	}
	stations := c.stationService.Search(query)
	c.shell.UpdateLastList(stations)
	ui.PrintStationTable(fmt.Sprintf("%s > %s", services.L.Get("cat_listing"), query), stations, "")
}

func (c *Commands) Istatistik(args []string) {
	top := c.statsService.GetTopStations(10)
	totalTime := c.statsService.GetTotalListenTime()
	sessions := c.statsService.GetTotalSessions()

	ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s: ", services.L.Get("stats_total_time"))
	ui.Fprintln(totalTime)
	ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s: ", services.L.Get("stats_total_sessions"))
	ui.Fprintln(sessions)

	if len(top) == 0 {
		ui.PrintInfo(services.L.Get("stats_no_data"))
		return
	}

	ui.PrintHeader(services.L.Get("stats_top_title"))
	for _, s := range top {
		ui.Fprintf("  %-30s | %d s | %d sessions\n", s.StationName, s.TotalSeconds, s.SessionCount)
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
	ui.Fprintf("  %-20s: %s\n", services.L.Get("sys_os"), stats["os"])
	ui.Fprintf("  %-20s: %v\n", "Go Version", stats["go_version"])
	ui.Fprintf("  %-20s: %.2f%%\n", services.L.Get("sys_cpu"), stats["cpu_percent"])
	ui.Fprintf("  %-20s: %s\n", services.L.Get("sys_total_mem"), c.systemService.FormatBytes(mem["total_memory"].(uint64)))
}

func (c *Commands) Clear(args []string) {
	ui.Fprint("\033[H\033[2J")
	ui.PrintBanner()
}

func (c *Commands) Web(args []string) {
	const url = "http://127.0.0.1:8765"

	c.webMu.Lock()
	if c.webStarted {
		c.webMu.Unlock()
		ui.PrintInfo(fmt.Sprintf("Web sunucusu zaten çalışıyor: %s", url))
		c.openBrowser(url)
		return
	}
	c.webStarted = true
	c.webMu.Unlock()

	server := web.NewWebServer(c.player, c.stationService, c.settingsService, c.systemService)
	go func() {
		if err := server.Start("127.0.0.1", 8765); err != nil {
			c.webMu.Lock()
			c.webStarted = false
			c.webMu.Unlock()
			ui.PrintError(fmt.Sprintf("Web server error: %v", err))
		}
	}()

	ui.PrintSuccess(fmt.Sprintf("Web sunucusu başlatılıyor: %s", url))
	time.AfterFunc(1500*time.Millisecond, func() { c.openBrowser(url) })
}

func (c *Commands) Cal(args []string) {
	fs := newFlagSet("cal")
	idFlag := fs.String("i", "", "station id")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: cal <id_or_index>")
		return
	}

	idOrIdx := firstNonEmpty(*idFlag)
	if idOrIdx == "" && len(fs.Args()) > 0 {
		idOrIdx = fs.Args()[0]
	}
	if idOrIdx == "" {
		ui.PrintError("Usage: cal <id_or_index>")
		return
	}

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

	c.playStation(*st)
}

func (c *Commands) Son(args []string) {
	lastID := c.settingsService.GetLastStationID()
	if lastID == "" {
		ui.PrintError(services.L.Get("msg_no_last_station"))
		return
	}

	st := c.stationService.GetStation(lastID)
	if st == nil {
		ui.PrintError(services.L.Get("msg_last_station_missing"))
		return
	}
	c.playStation(*st)
}

func (c *Commands) Dur(args []string) {
	c.recordSession()
	c.player.Stop()
	ui.PrintSuccess(services.L.Get("msg_stop_playing"))
}

func (c *Commands) Ses(args []string) {
	fs := newFlagSet("ses")
	levelShort := fs.Int("s", -1, "volume")
	levelLong := fs.Int("seviye", -1, "volume")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: ses -s <0-100>")
		return
	}

	level := *levelShort
	if *levelLong >= 0 {
		level = *levelLong
	}
	if level < 0 && len(fs.Args()) > 0 {
		vol, err := strconv.Atoi(fs.Args()[0])
		if err == nil {
			level = vol
		}
	}
	if level < 0 {
		ui.PrintInfo(fmt.Sprintf("%s: %%%d", services.L.Get("volume"), c.settingsService.GetVolume()))
		return
	}

	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	c.player.SetVolume(level, true)
	c.settingsService.SetVolume(level)
	c.settingsService.SetMuted(false)
	ui.PrintSuccess(services.L.Get("msg_vol_set", map[string]interface{}{"vol": level}))
}

func (c *Commands) Sessiz(args []string) {
	newMuted := !c.settingsService.IsMuted()
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "ac", "on", "1", "true", "evet":
			newMuted = true
		case "kapat", "off", "0", "false", "hayir", "hayır":
			newMuted = false
		default:
			ui.PrintError("Usage: sessiz [ac|kapat]")
			return
		}
	}

	c.player.SetMuted(newMuted)
	c.settingsService.SetMuted(newMuted)

	if newMuted {
		ui.PrintInfo(services.L.Get("msg_muted"))
	} else {
		ui.PrintInfo(services.L.Get("msg_unmuted"))
	}
}

func (c *Commands) adjacent(offset int) {
	lastList := c.shell.GetLastList()
	if len(lastList) == 0 {
		ui.PrintError(services.L.Get("msg_need_list"))
		return
	}

	current := c.currentStation()
	if current == nil {
		ui.PrintError(services.L.Get("msg_no_playing_station"))
		return
	}

	for i, st := range lastList {
		if st.ID == current.ID {
			nextIdx := (i + offset) % len(lastList)
			if nextIdx < 0 {
				nextIdx += len(lastList)
			}
			c.playStation(lastList[nextIdx])
			return
		}
	}
	ui.PrintError(services.L.Get("msg_station_not_in_list"))
}

func (c *Commands) Sonraki(args []string) {
	c.adjacent(1)
}

func (c *Commands) Onceki(args []string) {
	c.adjacent(-1)
}

func (c *Commands) Karistir(args []string) {
	fs := newFlagSet("karistir")
	country := fs.String("u", "", "country")
	countryLong := fs.String("ulke", "", "country")
	genre := fs.String("t", "", "genre")
	genreLong := fs.String("tur", "", "genre")
	if err := fs.Parse(args); err != nil {
		return
	}

	countryFilter := strings.ToLower(firstNonEmpty(*country, *countryLong))
	genreFilter := strings.ToLower(firstNonEmpty(*genre, *genreLong))

	var stations []models.RadioStation
	for _, st := range c.stationService.GetAllStations() {
		if countryFilter != "" && !strings.Contains(strings.ToLower(st.Country), countryFilter) {
			continue
		}
		if genreFilter != "" && !strings.Contains(strings.ToLower(st.Genre), genreFilter) {
			continue
		}
		stations = append(stations, st)
	}

	if len(stations) == 0 {
		ui.PrintError(services.L.Get("msg_no_match"))
		return
	}

	c.playStation(stations[c.rng.Intn(len(stations))])
}

func (c *Commands) Uyku(args []string) {
	if len(args) > 0 && strings.EqualFold(args[0], "iptal") {
		c.sleepMu.Lock()
		timer := c.sleepTimer
		c.sleepTimer = nil
		c.sleepMu.Unlock()

		if timer == nil {
			ui.PrintInfo(services.L.Get("msg_no_sleep"))
			return
		}
		timer.Stop()
		ui.PrintSuccess(services.L.Get("msg_sleep_cancel"))
		return
	}

	fs := newFlagSet("uyku")
	minutesShort := fs.Int("d", 0, "minutes")
	minutesLong := fs.Int("dakika", 0, "minutes")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: uyku -d <minutes> OR uyku iptal")
		return
	}

	minutes := *minutesShort
	if *minutesLong > 0 {
		minutes = *minutesLong
	}
	if minutes <= 0 && len(fs.Args()) > 0 {
		if parsed, err := strconv.Atoi(fs.Args()[0]); err == nil {
			minutes = parsed
		}
	}
	if minutes <= 0 {
		ui.PrintError("Usage: uyku -d <minutes> OR uyku iptal")
		return
	}

	c.sleepMu.Lock()
	if c.sleepTimer != nil {
		c.sleepTimer.Stop()
	}
	c.sleepTimer = time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
		c.sleepMu.Lock()
		c.sleepTimer = nil
		c.sleepMu.Unlock()
		c.Dur(nil)
		ui.PrintInfo(services.L.Get("msg_sleep_done"))
	})
	c.sleepMu.Unlock()

	ui.PrintSuccess(services.L.Get("msg_sleep_set", map[string]interface{}{"min": minutes}))
}

func (c *Commands) Gecmis(args []string) {
	c.songMu.Lock()
	history := append([]string(nil), c.songHistory...)
	c.songMu.Unlock()

	if len(history) == 0 {
		ui.PrintInfo(services.L.Get("msg_no_history"))
		return
	}

	ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s\n", services.L.Get("msg_recent_songs"))
	start := len(history) - 10
	if start < 0 {
		start = 0
	}
	no := 1
	for i := len(history) - 1; i >= start; i-- {
		ui.Fprintf("  %2d. %s\n", no, history[i])
		no++
	}
}

func (c *Commands) Favori(args []string) {
	fs := newFlagSet("favori")
	idFlag := fs.String("i", "", "station id")
	if err := fs.Parse(args); err != nil {
		ui.PrintError("Usage: favori [id]")
		return
	}

	id := firstNonEmpty(*idFlag)
	if id == "" && len(fs.Args()) > 0 {
		id = fs.Args()[0]
	}
	if id == "" {
		st := c.currentStation()
		if st == nil {
			ui.PrintError(services.L.Get("msg_no_playing_station"))
			return
		}
		id = st.ID
	}

	st := c.stationService.GetStation(id)
	if st == nil {
		ui.PrintError(services.L.Get("msg_station_not_found"))
		return
	}

	added := c.stationService.ToggleFavorite(st.ID)
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
	fs := newFlagSet("tema")
	nameFlag := fs.String("i", "", "theme")
	if err := fs.Parse(args); err != nil {
		return
	}

	name := firstNonEmpty(*nameFlag)
	if name == "" && len(fs.Args()) > 0 {
		name = fs.Args()[0]
	}
	if name == "" {
		ui.PrintInfo(fmt.Sprintf("Mevcut Temalar: %s", strings.Join(ui.GetThemes(), ", ")))
		return
	}

	if ui.SetTheme(name) {
		ui.PrintSuccess(fmt.Sprintf("Tema '%s' olarak ayarlandı.", name))
	} else {
		ui.PrintError(fmt.Sprintf("Geçersiz tema. Mevcut: %s", strings.Join(ui.GetThemes(), ", ")))
	}
}

func (c *Commands) Kontrol(args []string) {
	fs := newFlagSet("kontrol")
	idFlag := fs.String("i", "", "station id")
	if err := fs.Parse(args); err != nil {
		return
	}

	stationID := firstNonEmpty(*idFlag)
	if stationID == "" && len(fs.Args()) > 0 {
		stationID = fs.Args()[0]
	}

	var stations []models.RadioStation
	if stationID != "" {
		st := c.stationService.GetStation(stationID)
		if st != nil {
			stations = append(stations, *st)
		}
	} else {
		stations = c.stationService.GetAllStations()
	}

	if len(stations) == 0 {
		ui.PrintError("Kontrol edilecek istasyon bulunamadı.")
		return
	}

	ui.PrintInfo(services.L.Get("msg_checking", map[string]interface{}{"count": len(stations)}))
	success := 0
	for _, st := range stations {
		if c.stationActive(st) {
			success++
			if stationID != "" {
				ui.PrintSuccess(fmt.Sprintf("%s: %s", st.Name, services.L.Get("msg_active")))
			}
		} else if stationID != "" {
			ui.PrintError(fmt.Sprintf("%s: %s", st.Name, services.L.Get("msg_failed")))
		}
	}

	if stationID == "" {
		ui.PrintInfo(services.L.Get("msg_check_done", map[string]interface{}{"success": success, "total": len(stations)}))
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
	fs := newFlagSet("ekle")
	id := fs.String("id", "", "station id")
	name := fs.String("isim", "", "station name")
	country := fs.String("ulke", "Türkiye", "country")
	genre := fs.String("tur", "Çeşitli", "genre")
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

func (c *Commands) Duzenle(args []string) {
	fs := newFlagSet("duzenle")
	id := fs.String("id", "", "station id")
	name := fs.String("isim", "", "station name")
	country := fs.String("ulke", "", "country")
	genre := fs.String("tur", "", "genre")
	url := fs.String("url", "", "stream url")

	if err := fs.Parse(args); err != nil {
		ui.PrintError("Kullanım: duzenle --id <id> [--isim ..] [--url ..] vs.")
		return
	}
	if *id == "" {
		ui.PrintError("Kullanım: duzenle --id <id> [--isim ..] [--url ..] vs.")
		return
	}

	existing := c.stationService.GetCustomStation(*id)
	if existing == nil {
		ui.PrintError(services.L.Get("msg_custom_only"))
		return
	}

	updated := models.RadioStation{
		ID:      existing.ID,
		Name:    firstNonEmpty(*name, existing.Name),
		Country: firstNonEmpty(*country, existing.Country),
		Genre:   firstNonEmpty(*genre, existing.Genre),
		URL:     firstNonEmpty(*url, existing.URL),
	}
	if c.stationService.UpdateCustomStation(updated) {
		ui.PrintSuccess(services.L.Get("msg_station_updated"))
	} else {
		ui.PrintError(services.L.Get("msg_custom_only"))
	}
}

func (c *Commands) Sil(args []string) {
	fs := newFlagSet("sil")
	idFlag := fs.String("id", "", "station id")
	if err := fs.Parse(args); err != nil {
		return
	}

	id := firstNonEmpty(*idFlag)
	if id == "" && len(fs.Args()) > 0 {
		id = fs.Args()[0]
	}
	if id == "" {
		ui.PrintError("Usage: sil --id <id>")
		return
	}

	if c.stationService.RemoveCustomStation(id) {
		ui.PrintSuccess(services.L.Get("msg_station_deleted", map[string]interface{}{"id": id}))
	} else {
		ui.PrintError(services.L.Get("msg_station_not_found"))
	}
}

func (c *Commands) Iceaktar(args []string) {
	fs := newFlagSet("iceaktar")
	file := fs.String("d", "", "playlist file")
	fileLong := fs.String("dosya", "", "playlist file")
	country := fs.String("u", "İçe Aktarılan", "country")
	countryLong := fs.String("ulke", "", "country")
	genre := fs.String("t", "Karma", "genre")
	genreLong := fs.String("tur", "", "genre")
	prefix := fs.String("p", "", "prefix")
	prefixLong := fs.String("prefix", "", "prefix")
	if err := fs.Parse(args); err != nil {
		return
	}

	filePath := firstNonEmpty(*file, *fileLong)
	if filePath == "" {
		ui.PrintError("Usage: iceaktar -d <file>")
		return
	}

	countryName := firstNonEmpty(*countryLong, *country)
	genreName := firstNonEmpty(*genreLong, *genre)
	namePrefix := firstNonEmpty(*prefixLong, *prefix)

	count := c.stationService.ImportPlaylist(filePath, countryName, genreName, namePrefix)
	if count > 0 {
		ui.PrintSuccess(services.L.Get("msg_import_success", map[string]interface{}{"count": count}))
	} else {
		ui.PrintError(services.L.Get("msg_import_fail"))
	}
}

func (c *Commands) Bildirim(args []string) {
	enabled := !c.notificationSvc.IsEnabled()
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "ac", "on", "1", "true", "evet":
			enabled = true
		case "kapat", "off", "0", "false", "hayir", "hayır":
			enabled = false
		default:
			ui.PrintError("Usage: bildirim [ac|kapat]")
			return
		}
	}

	c.notificationSvc.SetEnabled(enabled)
	if enabled {
		ui.PrintSuccess(services.L.Get("msg_notify_on"))
	} else {
		ui.PrintSuccess(services.L.Get("msg_notify_off"))
	}
}

func (c *Commands) OnlineAra(args []string) {
	fs := newFlagSet("online-ara")
	queryShort := fs.String("s", "", "query")
	queryLong := fs.String("sorgu", "", "query")
	countryShort := fs.String("u", "", "country")
	countryLong := fs.String("ulke", "", "country")
	genreShort := fs.String("t", "", "genre")
	genreLong := fs.String("tur", "", "genre")
	limit := fs.Int("l", 15, "limit")
	limitLong := fs.Int("limit", 0, "limit")
	if err := fs.Parse(args); err != nil {
		return
	}

	query := firstNonEmpty(*queryShort, *queryLong, strings.Join(fs.Args(), " "))
	country := firstNonEmpty(*countryShort, *countryLong)
	genre := firstNonEmpty(*genreShort, *genreLong)
	if *limitLong > 0 {
		*limit = *limitLong
	}

	if query == "" && country == "" && genre == "" {
		ui.PrintError("En az bir arama kriteri (-s, -u, -t) girin.")
		return
	}

	ui.PrintInfo(services.L.Get("msg_searching"))
	results, err := c.radioBrowser.Search(query, country, genre, *limit)
	if err != nil {
		ui.PrintError(err.Error())
		return
	}
	if len(results) == 0 {
		ui.PrintError(services.L.Get("msg_station_not_found"))
		return
	}

	c.lastOnlineResults = results
	ui.PrintHeader(services.L.Get("msg_search_results"))
	for i, s := range results {
		ui.Fprintf("  %2d. [%s] %s (%s)\n", i+1, s.CountryDisplay(), s.Name, s.GenreDisplay())
	}
	ui.PrintInfo(services.L.Get("msg_online_add_hint"))
}

func (c *Commands) OnlineEkle(args []string) {
	fs := newFlagSet("online-ekle")
	n := 0
	fs.IntVar(&n, "n", 0, "index from search results")
	fs.IntVar(&n, "no", 0, "index from search results")
	if err := fs.Parse(args); err != nil {
		return
	}

	if n <= 0 || n > len(c.lastOnlineResults) {
		ui.PrintError("Geçersiz numara. Önce 'online-ara' kullanın.")
		return
	}

	os := c.lastOnlineResults[n-1]
	uuidPart := strings.TrimSpace(os.UUID)
	if len(uuidPart) > 8 {
		uuidPart = uuidPart[:8]
	}
	id := fmt.Sprintf("rb-%d", time.Now().UnixNano())
	if uuidPart != "" {
		id = "rb-" + uuidPart
	}
	st := models.RadioStation{
		ID:      id,
		Name:    firstNonEmpty(strings.TrimSpace(os.Name), "Bilinmeyen Radyo"),
		Country: os.CountryDisplay(),
		Genre:   os.GenreDisplay(),
		URL:     os.URL,
	}
	c.stationService.AddCustomStation(st)
	ui.PrintSuccess(fmt.Sprintf("İstasyon eklendi: %s (ID: %s)", st.Name, st.ID))
}

func (c *Commands) Dil(args []string) {
	fs := newFlagSet("dil")
	langShort := fs.String("i", "", "language")
	langLong := fs.String("isim", "", "language")
	if err := fs.Parse(args); err != nil {
		return
	}

	lang := firstNonEmpty(*langShort, *langLong)
	if lang == "" && len(fs.Args()) > 0 {
		lang = fs.Args()[0]
	}

	languages := services.L.GetLanguages()
	if lang == "" {
		codes := make([]string, 0, len(languages))
		for code := range languages {
			codes = append(codes, code)
		}
		sort.Strings(codes)

		ui.CurrentTheme.Primary.Fprintf(ui.Output, "%s:\n", services.L.Get("lang_select_title"))
		for _, code := range codes {
			ui.Fprintf("  %-4s - %s\n", code, languages[code])
		}
		ui.PrintInfo(fmt.Sprintf("Usage: lang -i <%s>", strings.Join(codes, "|")))
		return
	}

	name, ok := languages[lang]
	if !ok {
		ui.PrintError(fmt.Sprintf("Usage: lang -i <%s>", strings.Join(languageCodes(languages), "|")))
		return
	}

	c.settingsService.SetLanguage(lang)
	services.L.SetLanguage(lang)
	ui.PrintSuccess(services.L.Get("lang_updated", map[string]interface{}{"lang": name}))
}

func languageCodes(languages map[string]string) []string {
	codes := make([]string, 0, len(languages))
	for code := range languages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}
