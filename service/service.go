// Package service defines Relay's operations using the vocabulary from the
// product spec: encounters are recorded, the user tags, the AI suggests, the
// user confirms, and the graph explains itself.
//
// The split between tagging and suggesting is deliberate and load-bearing.
// Nothing on the AI path can produce a confirmed relationship, because the AI
// path has no function that returns one.
package service

import (
	"context"
	"time"

	"github.com/cef31/relay/domain"
)

// EncounterInput describes a meeting as it is being recorded.
type EncounterInput struct {
	Title          string
	OccurredAt     time.Time
	PlaceId        string
	ParticipantIds []string
	FileIds        []string
	Note           string
}

// TagInput is a relationship the user asserts directly.
type TagInput struct {
	FromId string
	ToId   string
	Type   domain.RelationshipType

	// EvidenceEventIds is optional. A tag stands on the user's word alone,
	// but citing events lets Explain give a richer answer.
	EvidenceEventIds []string
}

// Relay is the application surface.
type Relay interface {
	// Record stores an encounter as evidence. Events are append-only: once
	// recorded, an event is not edited, because later inferences cite it.
	//
	// An encounter requires at least two participants. A solo entry is a diary
	// entry, not evidence of a relationship, and Record rejects it.
	Record(ctx context.Context, in EncounterInput) (domain.Event, error)

	// Tag creates a relationship the user asserted directly. The result is
	// deterministic and confirmed: Status is StatusConfirmed and Confidence
	// is 1. No inference runs. Tag is the only path to a confirmed edge that
	// does not go through Confirm.
	//
	// A tag is not automatically bidirectional. Whether the tie also holds in
	// reverse, and under what label, comes from in.Type.Direction(): symmetric
	// types reverse to themselves, inverse types to in.Type.Inverse(), and
	// one-way types have no reverse at all. Whether that reverse is stored or
	// derived on read is still open; see docs/design-notes.md.
	Tag(ctx context.Context, in TagInput) (domain.Relationship, error)

	// Suggest proposes relationships inferred from recorded events. Every
	// returned relationship has Status StatusSuggested and a Confidence below
	// 1. Suggest never writes a confirmed edge and never overwrites one that
	// the user already confirmed or rejected.
	Suggest(ctx context.Context, personId string) ([]domain.Relationship, error)

	// Confirm accepts a suggestion, promoting it to StatusConfirmed.
	Confirm(ctx context.Context, relationshipId string) error

	// Reject declines a suggestion. Rejections are retained rather than
	// deleted, so that Suggest does not propose the same edge again.
	Reject(ctx context.Context, relationshipId string) error

	// Explain answers "how do I know this person" in natural language, citing
	// the events behind the relationship. It describes only recorded evidence.
	Explain(ctx context.Context, fromId, toId string) (string, error)

	// Search answers natural-language queries over the graph, such as
	// "everyone I met at the Seoul conference".
	Search(ctx context.Context, query string) ([]domain.Person, error)
}
