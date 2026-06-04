package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"sort"
	"strings"
	"sync"
	"time"
)

type StationStat struct {
	StationID    string `json:"stationId"`
	StationName  string `json:"stationName"`
	Country      string `json:"country"`
	Genre        string `json:"genre"`
	TotalSeconds int    `json:"totalSeconds"`
	SessionCount int    `json:"sessionCount"`
}

type StatisticsService struct {
	config *config.RadioConfig
	stats  map[string]StationStat
	mu     sync.RWMutex
}

const minSessionSeconds = 30

func NewStatisticsService(cfg *config.RadioConfig) *StatisticsService {
	ss := &StatisticsService{
		config: cfg,
		stats:  make(map[string]StationStat),
	}
	ss.Load()
	return ss
}

func (s *StatisticsService) RecordSession(station models.RadioStation, duration time.Duration) {
	seconds := int(duration.Seconds())
	if seconds < minSessionSeconds {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sid := station.ID
	if stat, ok := s.stats[sid]; ok {
		stat.TotalSeconds += seconds
		stat.SessionCount++
		s.stats[sid] = stat
	} else {
		s.stats[sid] = StationStat{
			StationID:    sid,
			StationName:  station.Name,
			Country:      station.Country,
			Genre:        station.Genre,
			TotalSeconds: seconds,
			SessionCount: 1,
		}
	}
	s.Save()
}

func (s *StatisticsService) GetTopStations(limit int) []StationStat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []StationStat
	for _, stat := range s.stats {
		all = append(all, stat)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].TotalSeconds != all[j].TotalSeconds {
			return all[i].TotalSeconds > all[j].TotalSeconds
		}
		return strings.ToLower(all[i].StationName) < strings.ToLower(all[j].StationName)
	})

	if limit > len(all) {
		limit = len(all)
	}
	return all[:limit]
}

func (s *StatisticsService) GetTotalListenTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, stat := range s.stats {
		total += stat.TotalSeconds
	}
	return time.Duration(total) * time.Second
}

func (s *StatisticsService) GetTotalSessions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, stat := range s.stats {
		total += stat.SessionCount
	}
	return total
}

func (s *StatisticsService) Load() {
	path := s.config.StatsFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	var list []StationStat
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, stat := range list {
		s.stats[stat.StationID] = stat
	}
}

func (s *StatisticsService) Save() {
	top := s.GetTopStations(len(s.stats))
	data, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(s.config.StatsFile), 0755)
	ioutil.WriteFile(s.config.StatsFile, data, 0644)
}
