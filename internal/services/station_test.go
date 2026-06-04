package services

import (
	"radio-shell/internal/config"
	"testing"
)

func TestStationService(t *testing.T) {
	cfg := config.NewRadioConfig()
	ss := NewStationService(cfg)
	err := ss.Init()
	if err != nil {
		t.Fatalf("Failed to init StationService: %v", err)
	}

	stations := ss.GetAllStations()
	if len(stations) == 0 {
		t.Error("No stations loaded")
	}

	tr := ss.GetStation("tr-trt-fm")
	if tr == nil {
		t.Error("Could not find TRT FM")
	} else if tr.Name != "TRT FM" {
		t.Errorf("Expected TRT FM, got %s", tr.Name)
	}
}
