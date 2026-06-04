package services

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"radio-shell/internal/config"
	"radio-shell/internal/models"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

//go:embed stations.json
var internalStationsFS embed.FS

type StationService struct {
	config         *config.RadioConfig
	stations       []models.RadioStation
	customStations []models.RadioStation
	favorites      map[string]struct{}
	mu             sync.RWMutex
}

func NewStationService(cfg *config.RadioConfig) *StationService {
	return &StationService{
		config:    cfg,
		favorites: make(map[string]struct{}),
	}
}

func (s *StationService) Init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadFavorites()
	s.loadInternalStations()
	s.loadCustomStations()
	return nil
}

func (s *StationService) GetAllStations() []models.RadioStation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []models.RadioStation
	all = append(all, s.stations...)
	all = append(all, s.customStations...)

	result := make([]models.RadioStation, len(all))
	for i, st := range all {
		_, fav := s.favorites[st.ID]
		result[i] = st.WithFavorite(fav)
	}
	return result
}

func (s *StationService) GetStation(idOrName string) *models.RadioStation {
	if idOrName == "" {
		return nil
	}

	all := s.GetAllStations()
	idOrName = strings.ToLower(idOrName)

	// Exact match ID
	for _, st := range all {
		if strings.ToLower(st.ID) == idOrName {
			return &st
		}
	}

	// Name match
	for _, st := range all {
		if strings.ToLower(st.Name) == idOrName {
			return &st
		}
	}

	// Partial match name
	for _, st := range all {
		if strings.Contains(strings.ToLower(st.Name), idOrName) {
			return &st
		}
	}

	return nil
}

func (s *StationService) ToggleFavorite(stationID string) bool {
	st := s.GetStation(stationID)
	if st == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	realID := st.ID
	added := false
	if _, ok := s.favorites[realID]; ok {
		delete(s.favorites, realID)
	} else {
		s.favorites[realID] = struct{}{}
		added = true
	}

	s.saveFavorites()
	return added
}

func (s *StationService) GetFavorites() []models.RadioStation {
	all := s.GetAllStations()
	var favs []models.RadioStation
	for _, st := range all {
		if st.Favorite {
			favs = append(favs, st)
		}
	}
	return favs
}

func (s *StationService) AddCustomStation(st models.RadioStation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.customStations {
		if existing.ID == st.ID {
			s.customStations = append(s.customStations[:i], s.customStations[i+1:]...)
			break
		}
	}
	s.customStations = append(s.customStations, st)
	s.saveCustomStations()
}

func (s *StationService) RemoveCustomStation(stationID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	initialLen := len(s.customStations)
	var newCustom []models.RadioStation
	for _, st := range s.customStations {
		if st.ID != stationID {
			newCustom = append(newCustom, st)
		}
	}
	s.customStations = newCustom

	if len(s.customStations) < initialLen {
		s.saveCustomStations()
		if _, ok := s.favorites[stationID]; ok {
			delete(s.favorites, stationID)
			s.saveFavorites()
		}
		return true
	}
	return false
}

func (s *StationService) ImportPlaylist(filePath, country, genre, prefix string) int {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return 0
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return 0
	}

	lines := strings.Split(string(content), "\n")
	count := 0
	name := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			parts := strings.SplitN(line, ",", 2)
			if len(parts) > 1 {
				name = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "http") {
			if name == "" {
				name = fmt.Sprintf("Imported %s", uuid.New().String()[:6])
			}
			if prefix != "" {
				name = fmt.Sprintf("%s %s", prefix, name)
			}

			// Simple ID generation
			safeID := ""
			for _, r := range strings.ToLower(name) {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					safeID += string(r)
				} else {
					safeID += "-"
				}
			}

			st := models.RadioStation{
				ID:      safeID,
				Name:    name,
				Country: country,
				Genre:   genre,
				URL:     line,
			}
			s.AddCustomStation(st)
			count++
			name = ""
		}
	}
	return count
}

func (s *StationService) GetCountries() []string {
	all := s.GetAllStations()
	countriesMap := make(map[string]struct{})
	for _, st := range all {
		if st.Country != "" {
			countriesMap[st.Country] = struct{}{}
		}
	}

	var countries []string
	for c := range countriesMap {
		countries = append(countries, c)
	}
	sort.Strings(countries)
	return countries
}

func (s *StationService) GetGenres() []string {
	all := s.GetAllStations()
	genresMap := make(map[string]struct{})
	for _, st := range all {
		if st.Genre != "" {
			parts := strings.Split(st.Genre, "/")
			for _, p := range parts {
				genresMap[strings.TrimSpace(p)] = struct{}{}
			}
		}
	}

	var genres []string
	for g := range genresMap {
		genres = append(genres, g)
	}
	sort.Strings(genres)
	return genres
}

func (s *StationService) Search(query string) []models.RadioStation {
	all := s.GetAllStations()
	if query == "" {
		return all
	}

	q := strings.ToLower(query)
	var result []models.RadioStation
	for _, st := range all {
		if strings.Contains(strings.ToLower(st.Name), q) ||
			strings.Contains(strings.ToLower(st.Country), q) ||
			strings.Contains(strings.ToLower(st.Genre), q) {
			result = append(result, st)
		}
	}
	return result
}

func (s *StationService) loadInternalStations() {
	data, err := internalStationsFS.ReadFile("stations.json")
	if err != nil {
		fmt.Printf("Error reading internal stations: %v\n", err)
		return
	}

	var list models.StationList
	if err := json.Unmarshal(data, &list); err != nil {
		fmt.Printf("Error unmarshaling internal stations: %v\n", err)
		return
	}
	s.stations = list.Stations
}

func (s *StationService) loadFavorites() {
	path := s.config.FavoritesFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	var favs []string
	if err := json.Unmarshal(data, &favs); err != nil {
		return
	}

	for _, id := range favs {
		s.favorites[id] = struct{}{}
	}
}

func (s *StationService) saveFavorites() {
	var favs []string
	for id := range s.favorites {
		favs = append(favs, id)
	}

	data, err := json.MarshalIndent(favs, "", "  ")
	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(s.config.FavoritesFile), 0755)
	ioutil.WriteFile(s.config.FavoritesFile, data, 0644)
}

func (s *StationService) loadCustomStations() {
	path := s.config.CustomStationsFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	var list models.StationList
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}
	s.customStations = list.Stations
}

func (s *StationService) saveCustomStations() {
	list := models.StationList{Stations: s.customStations}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(s.config.CustomStationsFile), 0755)
	ioutil.WriteFile(s.config.CustomStationsFile, data, 0644)
}
