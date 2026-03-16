package domain

type VideoState struct {
	URL       string        `json:"url"`
	UpdatedAt int64         `json:"updatedAt"`
	Events    []Event       `json:"events"`
	Games     []GameRange   `json:"games"`
	Settings  VideoSettings `json:"settings"`
}

type VideoSummary struct {
	URL         string `json:"url"`
	EventsCount int    `json:"eventsCount"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type Event struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	VideoTimeSec  int    `json:"videoTimeSec"`
	PlayerID      string `json:"playerId,omitempty"`
	IsHighlighted bool   `json:"isHighlighted,omitempty"`
}

type GameRange struct {
	ID       string `json:"id"`
	StartSec int    `json:"startSec"`
	EndSec   *int   `json:"endSec"`
}

type VideoSettings struct {
	GroupVisibility      map[string]bool `json:"groupVisibility"`
	EventVisibility      map[string]bool `json:"eventVisibility"`
	SelectedGameFilter   string          `json:"selectedGameFilter"`
	SelectedRosterFilter string          `json:"selectedRosterFilter"`
	ShowOnlyHighlights   bool            `json:"showOnlyHighlights"`
	Teams                []Team          `json:"teams"`
	SelectedPlayerID     string          `json:"selectedPlayerId"`
}

type Team struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Color   string   `json:"color"`
	Players []Player `json:"players"`
}

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
