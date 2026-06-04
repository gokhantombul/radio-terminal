package web

import (
	"fmt"
	"net/http"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type WebServer struct {
	player          *player.AudioPlayer
	stationService  *services.StationService
	settingsService *services.SettingsService
	systemService   *services.SystemService
}

func NewWebServer(p *player.AudioPlayer, ss *services.StationService, set *services.SettingsService, sys *services.SystemService) *WebServer {
	return &WebServer{
		player:          p,
		stationService:  ss,
		settingsService: set,
		systemService:   sys,
	}
}

func (ws *WebServer) Start(host string, port int) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

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
		api.GET("/system", ws.getSystemInfo)
		api.GET("/language", ws.getLanguage)
		api.POST("/language/:lang_code", ws.setLanguage)
	}

	// Static files
	r.Static("/", "./internal/web/static")

	return r.Run(fmt.Sprintf("%s:%d", host, port))
}

func (ws *WebServer) getStations(c *gin.Context) {
	stations := ws.stationService.GetAllStations()
	c.JSON(http.StatusOK, stations)
}

func (ws *WebServer) getStatus(c *gin.Context) {
	station, song, vol, muted, playing, recording, elapsed := ws.player.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"is_playing":      playing,
		"current_station": station,
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
	c.JSON(http.StatusOK, gin.H{"status": "volume_set", "level": level})
}

func (ws *WebServer) setMute(c *gin.Context) {
	mutedStr := c.Param("muted")
	muted, _ := strconv.ParseBool(mutedStr)
	ws.player.SetMuted(muted)
	ws.settingsService.SetMuted(muted)
	c.JSON(http.StatusOK, gin.H{"status": "muted_set", "is_muted": muted})
}

func (ws *WebServer) toggleFavorite(c *gin.Context) {
	id := c.Param("station_id")
	added := ws.stationService.ToggleFavorite(id)
	c.JSON(http.StatusOK, gin.H{"status": "success", "is_favorite": added})
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
	ws.settingsService.SetLanguage(langCode)
	services.L.SetLanguage(langCode)
	c.JSON(http.StatusOK, gin.H{"status": "success", "language": langCode})
}
