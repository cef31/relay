package domain

// AssertionStatus tracks confirmation of any claim about the graph. The AI may
// suggest, but only the user promotes a claim to Confirmed.
//
// It is deliberately not named after any one kind of claim. Relationships use
// it today; attributes and facts extracted from files will use the same three
// states. One shared type means "AI suggests, user confirms" is expressed once
// rather than re-implemented, and cannot drift between them.
type AssertionStatus string

const (
	StatusSuggested AssertionStatus = "SUGGESTED"
	StatusConfirmed AssertionStatus = "CONFIRMED"
	StatusRejected  AssertionStatus = "REJECTED"
)
