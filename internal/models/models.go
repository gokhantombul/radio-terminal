package models

type RadioStation struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	Genre    string `json:"genre"`
	URL      string `json:"url"`
	Favorite bool   `json:"favorite"`
}

func (s RadioStation) WithFavorite(fav bool) RadioStation {
	s.Favorite = fav
	return s
}

type UserSettings struct {
	Volume               int     `json:"volume"`
	LastStationID        *string `json:"lastStationId"`
	NotificationsEnabled bool    `json:"notificationsEnabled"`
	Language             string  `json:"language"`
	Muted                bool    `json:"muted"`
}

func UserSettingsDefaults() UserSettings {
	return UserSettings{
		Volume:               100,
		LastStationID:        nil,
		NotificationsEnabled: true,
		Language:             "en",
		Muted:                false,
	}
}

type StationList struct {
	Stations []RadioStation `json:"stations"`
}
