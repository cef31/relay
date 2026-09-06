package domain

import "time"

// Coordinates is a point on the map. It is a separate type so that Place can
// hold it by pointer: a zero Latitude and Longitude is a real location in the
// Gulf of Guinea, and must not be confused with "location unknown".
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// Place is a location where events occurred.
type Place struct {
	Id   string
	Name string

	// Coordinates is nil when the location is unknown. It is captured at the
	// moment of an encounter so places can be shown on a map. Relay stores
	// points where meetings happened, not a continuous location history.
	Coordinates *Coordinates

	// ParentId optionally nests places: Hall B -> COEX -> Seoul. Searching a
	// broad place has to reach events tagged to a narrow one, or "everyone I
	// met at the Seoul conference" misses everything recorded as the hall.
	ParentId string

	CreatedAt time.Time
}
