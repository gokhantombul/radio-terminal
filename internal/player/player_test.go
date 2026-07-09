package player

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"radio-shell/internal/services"
)

func testPlayer(t *testing.T, command string) *AudioPlayer {
	t.Helper()
	appDir := t.TempDir()
	cfg := &config.RadioConfig{
		Player:             config.PlayerConfig{Command: command},
		AppDir:             appDir,
		FavoritesFile:      filepath.Join(appDir, "favorites.json"),
		CustomStationsFile: filepath.Join(appDir, "custom-stations.json"),
		SettingsFile:       filepath.Join(appDir, "settings.json"),
		RecordingsDir:      filepath.Join(appDir, "recordings"),
		StatsFile:          filepath.Join(appDir, "stats.json"),
		WebPIDFile:         filepath.Join(appDir, "web.pid"),
	}
	return NewAudioPlayer(cfg, services.NewNotificationService(services.NewSettingsService(cfg)))
}

// A player process that exits on its own must be reaped and reported as not
// playing. Previously Wait was never called for natural exits, so IsPlaying
// stayed true forever and the watchdog never restarted anything.
func TestDeadPlayerProcessIsReaped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses unix 'true' binary as a fake player")
	}
	p := testPlayer(t, "true")
	defer p.Stop()

	p.Play(models.RadioStation{ID: "x", Name: "X", URL: "http://example.invalid/stream"}, 50, false)

	// 'true' exits immediately; stay under the watchdog's first restart (~3s).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !p.IsPlaying() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("player still reports playing after its process exited")
}

func TestStopWithoutPlayIsSafe(t *testing.T) {
	p := testPlayer(t, "true")
	p.Stop()
	p.Stop()
	if p.IsPlaying() {
		t.Fatal("expected not playing")
	}
	if _, err := p.StartRecording(); err == nil {
		t.Fatal("expected StartRecording to fail when not playing")
	}
	if path := p.StopRecording(); path != "" {
		t.Fatalf("expected empty recording path, got %q", path)
	}
}
