package services

import (
	"radio-shell/internal/models"
	"testing"
	"time"
)

func TestStatisticsServiceRecordSessionPersistsWithoutDeadlock(t *testing.T) {
	stats := NewStatisticsService(testConfig(t))
	station := models.RadioStation{
		ID:      "test-station",
		Name:    "Test Station",
		Country: "Türkiye",
		Genre:   "Pop",
		URL:     "https://example.test/stream",
	}

	done := make(chan struct{})
	go func() {
		stats.RecordSession(station, 31*time.Second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("recording a listen session deadlocked")
	}

	if got := stats.GetTotalSessions(); got != 1 {
		t.Fatalf("expected one session, got %d", got)
	}

	top := stats.GetTopStations(1)
	if len(top) != 1 || top[0].StationID != station.ID || top[0].TotalSeconds != 31 {
		t.Fatalf("unexpected top stations: %+v", top)
	}
}
