package domain

import "time"

// Event is a meeting or encounter. Events are the factual record that
// relationships are inferred from, so they are never rewritten by the AI.
type Event struct {
	Id         string
	Title      string
	OccurredAt time.Time

	// PlaceId is the Place the event happened at, empty if unknown.
	PlaceId string
	// ParticipantIds are the Persons present, including the user.
	ParticipantIds []string
	// FileIds are the Files exchanged during the event.
	FileIds []string

	Note      string
	CreatedAt time.Time
}

// Involves reports whether personId took part in the event.
func (e Event) Involves(personId string) bool {
	for _, id := range e.ParticipantIds {
		if id == personId {
			return true
		}
	}
	return false
}
