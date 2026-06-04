package services

import (
	"radio-shell/internal/models"
	"testing"
)

func TestStationServiceUpdatesOnlyCustomStations(t *testing.T) {
	ss := NewStationService(testConfig(t))
	if err := ss.Init(); err != nil {
		t.Fatalf("init station service: %v", err)
	}

	custom := models.RadioStation{
		ID:      "custom-one",
		Name:    "Custom One",
		Country: "Türkiye",
		Genre:   "Rock",
		URL:     "https://example.test/stream",
	}
	ss.AddCustomStation(custom)

	found := ss.GetCustomStation("CUSTOM-ONE")
	if found == nil {
		t.Fatal("expected custom station to be found case-insensitively")
	}
	if found.Name != custom.Name {
		t.Fatalf("expected %q, got %q", custom.Name, found.Name)
	}

	custom.Name = "Updated Custom"
	custom.Genre = "Jazz"
	if !ss.UpdateCustomStation(custom) {
		t.Fatal("expected custom station update to succeed")
	}

	updated := ss.GetCustomStation("custom-one")
	if updated == nil {
		t.Fatal("expected updated custom station to exist")
	}
	if updated.Name != "Updated Custom" || updated.Genre != "Jazz" {
		t.Fatalf("unexpected updated station: %+v", updated)
	}

	if ss.UpdateCustomStation(models.RadioStation{ID: "tr-trt-fm", Name: "Nope"}) {
		t.Fatal("built-in stations must not be editable as custom stations")
	}
}
