package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"sync"
)

type SettingsService struct {
	config   *config.RadioConfig
	settings models.UserSettings
	mu       sync.RWMutex
}

func NewSettingsService(cfg *config.RadioConfig) *SettingsService {
	ss := &SettingsService{
		config:   cfg,
		settings: models.UserSettingsDefaults(),
	}
	ss.Load()
	return ss
}

func (s *SettingsService) GetVolume() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Volume
}

func (s *SettingsService) SetVolume(volume int) {
	s.mu.Lock()
	s.settings.Volume = volume
	s.mu.Unlock()
	s.Save()
}

func (s *SettingsService) IsMuted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Muted
}

func (s *SettingsService) SetMuted(muted bool) {
	s.mu.Lock()
	s.settings.Muted = muted
	s.mu.Unlock()
	s.Save()
}

func (s *SettingsService) GetLastStationID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.settings.LastStationID == nil {
		return ""
	}
	return *s.settings.LastStationID
}

func (s *SettingsService) SetLastStationID(stationID string) {
	s.mu.Lock()
	s.settings.LastStationID = &stationID
	s.mu.Unlock()
	s.Save()
}

func (s *SettingsService) IsNotificationsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.NotificationsEnabled
}

func (s *SettingsService) SetNotificationsEnabled(enabled bool) {
	s.mu.Lock()
	s.settings.NotificationsEnabled = enabled
	s.mu.Unlock()
	s.Save()
}

func (s *SettingsService) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Language
}

func (s *SettingsService) SetLanguage(language string) {
	s.mu.Lock()
	s.settings.Language = language
	s.mu.Unlock()
	s.Save()
}

func (s *SettingsService) Load() {
	path := s.config.SettingsFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	json.Unmarshal(data, &s.settings)
}

func (s *SettingsService) Save() {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.settings, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(s.config.SettingsFile), 0755)
	ioutil.WriteFile(s.config.SettingsFile, data, 0644)
}
