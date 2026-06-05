package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"radio-shell/internal/config"
	"radio-shell/internal/player"
	"radio-shell/internal/services"
)

func TestRouterServesAPIAndStaticFiles(t *testing.T) {
	ws := testWebServer(t)
	router := ws.router()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "api status", path: "/api/status", wantStatus: http.StatusOK, wantBody: "is_playing"},
		{name: "index", path: "/", wantStatus: http.StatusOK, wantBody: "Radio Shell"},
		{name: "stylesheet", path: "/style.css", wantStatus: http.StatusOK, wantBody: "body"},
		{name: "api not found", path: "/api/missing", wantStatus: http.StatusNotFound, wantBody: "not found"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("expected body to contain %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func testWebServer(t *testing.T) *WebServer {
	t.Helper()

	appDir := t.TempDir()
	cfg := &config.RadioConfig{
		Player: config.PlayerConfig{
			Command: "ffplay",
			Args:    []string{"-nodisp", "-hide_banner", "-loglevel", "quiet", "-autoexit"},
		},
		AppDir:             appDir,
		FavoritesFile:      filepath.Join(appDir, "favorites.json"),
		CustomStationsFile: filepath.Join(appDir, "custom-stations.json"),
		SettingsFile:       filepath.Join(appDir, "settings.json"),
		RecordingsDir:      filepath.Join(appDir, "recordings"),
		StatsFile:          filepath.Join(appDir, "stats.json"),
		WebPIDFile:         filepath.Join(appDir, "web.pid"),
	}
	settings := services.NewSettingsService(cfg)
	stations := services.NewStationService(cfg)
	if err := stations.Init(); err != nil {
		t.Fatalf("init stations: %v", err)
	}
	notifications := services.NewNotificationService(settings)
	audioPlayer := player.NewAudioPlayer(cfg, notifications)
	system := services.NewSystemService()
	return NewWebServer(audioPlayer, stations, settings, system)
}
