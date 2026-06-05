package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"radio-shell/internal/models"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

type WebServer struct {
	player          *player.AudioPlayer
	stationService  *services.StationService
	settingsService *services.SettingsService
	systemService   *services.SystemService
}

type stationInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Genre      string `json:"genre"`
	Country    string `json:"country"`
	IsFavorite bool   `json:"is_favorite"`
}

func NewWebServer(p *player.AudioPlayer, ss *services.StationService, set *services.SettingsService, sys *services.SystemService) *WebServer {
	if !p.IsPlaying() {
		p.SetVolume(set.GetVolume(), false)
		p.SetMuted(set.IsMuted())
	}

	return &WebServer{
		player:          p,
		stationService:  ss,
		settingsService: set,
		systemService:   sys,
	}
}

func (ws *WebServer) Start(host string, port int) error {
	r := ws.router()
	return r.Run(fmt.Sprintf("%s:%d", host, port))
}

func (ws *WebServer) router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// API Routes
	api := r.Group("/api")
	{
		api.GET("/stations", ws.getStations)
		api.GET("/status", ws.getStatus)
		api.POST("/play/:station_id", ws.playStation)
		api.POST("/stop", ws.stopPlayback)
		api.POST("/volume/:level", ws.setVolume)
		api.POST("/mute/:muted", ws.setMute)
		api.POST("/favorite/:station_id", ws.toggleFavorite)
		api.POST("/record/start", ws.startRecording)
		api.POST("/record/stop", ws.stopRecording)
		api.GET("/system", ws.getSystemInfo)
		api.GET("/language", ws.getLanguage)
		api.POST("/language/:lang_code", ws.setLanguage)
		api.GET("/locales", ws.getLocales)
	}

	r.GET("/", func(c *gin.Context) {
		data, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "index not found"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		http.FileServer(staticFileSystem()).ServeHTTP(c.Writer, c.Request)
	})

	return r
}

func staticFileSystem() http.FileSystem {
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(static)
}

func (ws *WebServer) getStations(c *gin.Context) {
	stations := ws.stationService.GetAllStations()
	result := make([]stationInfo, 0, len(stations))
	for _, st := range stations {
		result = append(result, toStationInfo(st))
	}
	c.JSON(http.StatusOK, result)
}

func (ws *WebServer) getStatus(c *gin.Context) {
	station, song, vol, muted, playing, recording, elapsed := ws.player.GetStatus()
	var currentStation *stationInfo
	if station != nil {
		info := toStationInfo(*station)
		if fresh := ws.stationService.GetStation(station.ID); fresh != nil {
			info.IsFavorite = fresh.Favorite
		}
		currentStation = &info
	}

	c.JSON(http.StatusOK, gin.H{
		"is_playing":      playing,
		"current_station": currentStation,
		"current_song":    song,
		"volume":          vol,
		"is_muted":        muted,
		"is_recording":    recording,
		"elapsed_seconds": elapsed,
	})
}

func (ws *WebServer) playStation(c *gin.Context) {
	id := c.Param("station_id")
	st := ws.stationService.GetStation(id)
	if st == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Station not found"})
		return
	}

	ws.player.Play(*st, ws.settingsService.GetVolume(), ws.settingsService.IsMuted())
	ws.settingsService.SetLastStationID(st.ID)
	c.JSON(http.StatusOK, gin.H{"status": "playing", "station": st.Name})
}

func (ws *WebServer) stopPlayback(c *gin.Context) {
	ws.player.Stop()
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (ws *WebServer) setVolume(c *gin.Context) {
	levelStr := c.Param("level")
	level, err := strconv.Atoi(levelStr)
	if err != nil || level < 0 || level > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid volume level"})
		return
	}

	ws.settingsService.SetVolume(level)
	if level > 0 {
		ws.settingsService.SetMuted(false)
	}
	ws.player.SetVolume(level, true)
	_, _, _, muted, _, _, _ := ws.player.GetStatus()
	c.JSON(http.StatusOK, gin.H{"status": "volume_set", "level": level, "is_muted": muted})
}

func (ws *WebServer) setMute(c *gin.Context) {
	mutedStr := c.Param("muted")
	muted, _ := strconv.ParseBool(mutedStr)
	ws.player.SetMuted(muted)
	ws.settingsService.SetMuted(muted)
	status := "unmuted"
	if muted {
		status = "muted"
	}
	c.JSON(http.StatusOK, gin.H{"status": status, "is_muted": muted})
}

func (ws *WebServer) toggleFavorite(c *gin.Context) {
	id := c.Param("station_id")
	st := ws.stationService.GetStation(id)
	if st == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Station not found"})
		return
	}
	added := ws.stationService.ToggleFavorite(st.ID)
	c.JSON(http.StatusOK, gin.H{"status": "success", "is_favorite": added})
}

func (ws *WebServer) startRecording(c *gin.Context) {
	file, err := ws.player.StartRecording()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": recordingErrorMessage(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": services.L.Get("msg_recording_started", map[string]interface{}{"file": file}),
	})
}

func (ws *WebServer) stopRecording(c *gin.Context) {
	path := ws.player.StopRecording()
	if path == "" {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": services.L.Get("msg_no_active_record")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": services.L.Get("msg_recording_stopped", map[string]interface{}{"path": path}),
	})
}

func (ws *WebServer) getSystemInfo(c *gin.Context) {
	c.JSON(http.StatusOK, ws.systemService.GetWebInfo())
}

func (ws *WebServer) getLanguage(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"current":   ws.settingsService.GetLanguage(),
		"available": services.L.GetLanguages(),
	})
}

func (ws *WebServer) setLanguage(c *gin.Context) {
	langCode := c.Param("lang_code")
	if _, ok := services.L.GetLanguages()[langCode]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language not supported"})
		return
	}
	ws.settingsService.SetLanguage(langCode)
	services.L.SetLanguage(langCode)
	c.JSON(http.StatusOK, gin.H{"status": "success", "language": langCode})
}

func (ws *WebServer) getLocales(c *gin.Context) {
	c.JSON(http.StatusOK, services.L.GetStrings())
}

func toStationInfo(st models.RadioStation) stationInfo {
	return stationInfo{
		ID:         st.ID,
		Name:       st.Name,
		Genre:      st.Genre,
		Country:    st.Country,
		IsFavorite: st.Favorite,
	}
}

func recordingErrorMessage(err error) string {
	switch err.Error() {
	case "not playing":
		return services.L.Get("msg_not_playing")
	case "already recording":
		return services.L.Get("msg_already_recording")
	default:
		return services.L.Get("msg_recording_failed", map[string]interface{}{"error": err})
	}
}
