package domain

import "time"

// RelationshipType labels an edge between two people.
type RelationshipType string

const (
	RelationshipFriend    RelationshipType = "FRIEND"
	RelationshipFamily    RelationshipType = "FAMILY"
	RelationshipTeacher   RelationshipType = "TEACHER"
	RelationshipStudent   RelationshipType = "STUDENT"
	RelationshipColleague RelationshipType = "COLLEAGUE"
)

// Direction describes how a relationship type reads from the other side.
type Direction string

const (
	// DirectionSymmetric: the reverse carries the same label — FRIEND/FRIEND.
	DirectionSymmetric Direction = "SYMMETRIC"
	// DirectionInverse: the reverse carries a different label — TEACHER/STUDENT.
	DirectionInverse Direction = "INVERSE"
	// DirectionOneWay: there is no reverse. The tie holds only as recorded,
	// as when the user remembers someone who does not know them.
	DirectionOneWay Direction = "ONE_WAY"
)

// directions maps each type to how it reverses. A type absent from this table
// is treated as one-way, so a newly added type never silently invents a
// reverse edge it was never given one for.
var directions = map[RelationshipType]struct {
	direction Direction
	inverse   RelationshipType
}{
	RelationshipFriend:    {DirectionSymmetric, RelationshipFriend},
	RelationshipColleague: {DirectionSymmetric, RelationshipColleague},
	RelationshipFamily:    {DirectionSymmetric, RelationshipFamily},
	RelationshipTeacher:   {DirectionInverse, RelationshipStudent},
	RelationshipStudent:   {DirectionInverse, RelationshipTeacher},
}

// Direction reports how t reads from the other person's side.
func (t RelationshipType) Direction() Direction {
	d, ok := directions[t]
	if !ok {
		return DirectionOneWay
	}
	return d.direction
}

// Inverse returns the label the tie carries in the reverse direction. ok is
// false for one-way types, which have no reverse edge to derive.
func (t RelationshipType) Inverse() (RelationshipType, bool) {
	d, ok := directions[t]
	if !ok || d.direction == DirectionOneWay {
		return "", false
	}
	return d.inverse, true
}

// Relationship is a directed edge from one Person to another.
type Relationship struct {
	Id     string
	FromId string
	ToId   string
	Type   RelationshipType
	Status AssertionStatus

	// Confidence is the AI's certainty in Type, from 0 to 1.
	// It is 1 for edges the user created directly.
	Confidence float64

	// EvidenceEventIds are the Events that support this relationship.
	// It is what lets Relay explain how two people came to be connected.
	EvidenceEventIds []string

	CreatedAt time.Time
	// ConfirmedAt is zero until Status becomes StatusConfirmed.
	ConfirmedAt time.Time
}

// IsConfirmed reports whether the user has accepted this relationship.
func (r Relationship) IsConfirmed() bool {
	return r.Status == StatusConfirmed
}
