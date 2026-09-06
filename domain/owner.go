package domain

// Owner identifies which Person this graph belongs to. Exactly one Owner
// exists, and it is a singleton rather than a flag on Person so that the
// "exactly one" invariant is structural: a boolean spread across every Person
// row can silently become zero after an import or two after a merge, and both
// break traversal without any obvious failure.
//
// The owner is an ordinary Person in every other respect. That is deliberate —
// the owner's own name, birth date, and facts reuse the same machinery as
// everyone else's, instead of a parallel profile type.
//
// The owner node carries three rules that ordinary people do not:
//   - it cannot be deleted
//   - it cannot be merged into another Person
//   - it is never an endpoint of a suggested edge to itself
type Owner struct {
	PersonId string
}

// IsOwner reports whether personId is the owner of this graph.
func (o Owner) IsOwner(personId string) bool {
	return o.PersonId != "" && o.PersonId == personId
}
