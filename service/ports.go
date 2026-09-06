package service

import (
	"context"

	"github.com/cef31/relay/domain"
)

// The ports below are declared here, in the package that consumes them, rather
// than in a storage package. The dependency then points inward: adapters
// import service, service imports nobody.

// Repo is persistence for the graph — writes and point reads by id.
//
// Deliberately not a generic CRUD layer over rows: Suggest and Search need
// traversal, which lives in GraphQuery instead. Answering "everyone two hops
// from the owner" through GetPerson would mean loading the graph into memory
// and filtering in Go.
type Repo interface {
	GetOwner(ctx context.Context) (domain.Owner, error)
	SetOwner(ctx context.Context, o domain.Owner) error

	GetPerson(ctx context.Context, id string) (domain.Person, error)
	SavePerson(ctx context.Context, p domain.Person) error

	GetPlace(ctx context.Context, id string) (domain.Place, error)
	SavePlace(ctx context.Context, p domain.Place) error

	GetEvent(ctx context.Context, id string) (domain.Event, error)
	SaveEvent(ctx context.Context, e domain.Event) error

	GetRelationship(ctx context.Context, id string) (domain.Relationship, error)
	SaveRelationship(ctx context.Context, r domain.Relationship) error
}

// Store adds the transaction boundary to Repo.
//
// Record followed by Tag must not half-commit: a relationship citing an event
// that failed to write is corrupt evidence, and Explain would later narrate
// from it. Multi-write operations run inside Atomically.
type Store interface {
	Repo

	// Atomically runs fn against a Repo bound to one transaction, committing
	// if fn returns nil and rolling back otherwise. The Repo passed to fn must
	// not be retained after it returns.
	Atomically(ctx context.Context, fn func(Repo) error) error
}

// GraphQuery is read-only traversal. Suggest and Search are graph-shaped
// questions, not row lookups.
type GraphQuery interface {
	// CoOccurrences returns events where personId and at least one other
	// person both participated. This is the raw material for inferring
	// relationships from shared history.
	CoOccurrences(ctx context.Context, personId string) ([]domain.Event, error)

	// Neighbors returns relationships with personId at either endpoint.
	// Both endpoints, because the graph is general: reaching "father's friend"
	// means traversing edges the owner is not part of.
	Neighbors(ctx context.Context, personId string) ([]domain.Relationship, error)

	// PlacesUnder returns placeId together with every place nested beneath it.
	// Without this, a search for Seoul misses everything recorded against the
	// specific hall.
	PlacesUnder(ctx context.Context, placeId string) ([]string, error)

	// EventsAtPlaces returns events recorded at any of the given places.
	EventsAtPlaces(ctx context.Context, placeIds []string) ([]domain.Event, error)
}

// Match is one semantic search hit.
type Match struct {
	EntityId string
	Score    float64
}

// SemanticIndex ranks entities by similarity to free-text queries. It exists
// because structured queries cannot reach "that friendly designer I met last
// spring" — no field holds "friendly" or "designer".
//
// It is a port rather than part of domain because an embedding is an
// infrastructure detail, not a fact about a person.
type SemanticIndex interface {
	// Index stores or replaces the vector for one entity.
	Index(ctx context.Context, entityId, text string) error

	// Similar ranks candidates by similarity to query. Candidates come from a
	// GraphQuery, which is the hybrid split: the graph decides what is
	// eligible, the index only orders it. Passing nil searches everything,
	// which is right for open-ended recall and wrong for any query with a
	// structured part — embeddings blur "Seoul" toward "Busan".
	//
	// Results are entity ids, never text. Explain narrates from the stored
	// records; narrating from indexed text would fabricate.
	Similar(ctx context.Context, query string, candidates []string) ([]Match, error)

	// ModelId identifies the embedding model behind the current vectors.
	// Vectors from different models are not comparable, so a changed id means
	// the whole index must be rebuilt. Without it, a model upgrade silently
	// returns nonsense.
	ModelId(ctx context.Context) (string, error)
}
