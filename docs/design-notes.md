# Design notes

Decisions and open questions that the code does not yet express.

## Search is hybrid: graph filters, vectors rank

Natural-language search splits into two query shapes, and neither mechanism
handles both:

- *"Everyone I met at the Seoul conference"* — structured. Place containment
  plus participant lookup. Vector search answers this **worse** than
  traversal: embeddings blur "Seoul" toward "Busan" and "conference" toward
  "meetup", returning plausible wrong people with no signal that they are
  wrong.
- *"That friendly designer I met last spring"* — semantic. No structured field
  holds "friendly" or "designer".

**Decision: the graph filters, the index ranks.** `GraphQuery` narrows to
eligible candidates (place subtree, participants, date range); `SemanticIndex`
orders within that set. The split is visible in the signature —
`Similar(ctx, query, candidates)` takes the candidate set as an argument.

Retrieval returns **entity ids, never text**. `Explain` narrates from the
stored records. Narrating from indexed text is fabrication with extra steps.

### Implementation constraints

- **No vector database.** A personal graph is hundreds to a few thousand
  entities. Brute-force cosine over ~5,000 vectors is about a millisecond on a
  phone. No ANN index, no `sqlite-vec`. This stays true for years of use.
- **Embed on device.** A hosted embedding API would ship the text of every
  note off the device, contradicting local-by-default. Cost is APK size and
  latency for a small on-device model; that tradeoff is the real decision.
- **Version the vectors.** Every vector carries the id of the model that
  produced it (`SemanticIndex.ModelId`). Vectors from different models are not
  comparable, so a model upgrade requires a full re-embed. Detectable by
  version; silently broken without one.

## Attributes are time-relative — and time is a filter, not a ranking

Attributes carry `ValidFrom` / `ValidUntil`, so every read needs an *as-of*
time:

```go
// AttributesAsOf returns the attributes true for a person at an instant:
// ValidFrom <= at && (ValidUntil.IsZero() || at < ValidUntil).
AttributesAsOf(ctx, personId string, at time.Time) ([]Attribute, error)
```

"Where do they live?" passes `time.Now()`. "Where did they live when we met?"
passes that event's `OccurredAt` — the question Relay exists to answer, and it
falls straight out of the interval check.

**Vector search cannot do this.** Considered and rejected:

- **Embeddings have no arithmetic.** "2019" and "2020" embed almost
  identically. Given `Seoul, valid 2015–2018` and the query "where did they
  live in 2019", similarity is *high* — same person, same city, adjacent year.
  Nothing in cosine distance evaluates `2019 ∈ [2015, 2018]`. Interval
  containment is a comparison, not a distance.
- **Similarity favors the wrong answer.** The nearest neighbour to "their
  address in 2019" may be the 2023 address, sharing more phrasing. The result
  is a false fact ranked first with no signal it is stale, and `Explain` then
  narrates a present-tense fabrication assembled entirely from real records.

This is the hybrid rule above, applied: time is structured and totally
ordered, so it belongs in the filter, evaluated before anything is ranked.

### Interval rules

- **Zero `ValidFrom` means unbounded past**, not invalid. Most facts arrive
  with no known start date and would otherwise match nothing.
- **Zero `ValidUntil` means still true.**
- **`ValidUntil` is usually when you *found out*, not when the fact changed.**
  You learn the new address, not the moving date. The intervals are soft at
  the edges even though the comparison against them is exact — worth
  remembering before trusting a boundary date.

### Where vectors do help attributes

Not for time, but for reaching the right attribute from vague language:
matching paraphrased values ("SNU" and "Seoul National University" are one
school), and interpreting "that guy who studied where I did". Both are ranking
problems over a candidate set the temporal filter already narrowed.

## Relay is not a diary

Relationships involve two or more people. Relay records *encounters between
people*, not what the user did.

**Rule: an encounter requires at least two participants.** `Record` rejects
fewer. An event containing only the owner is a journal entry — it is evidence
of no relationship, no inference can use it, and allowing it lets the product
drift into a diary with a graph attached.

Consequence for accuracy: the primary write path is live capture, with both
people present. `OccurredAt` and `PlaceId` are observed, not remembered, so
their accuracy is bounded by the device rather than by memory.

This is why `OccurredAt` needs no precision modeling (year/month/day
granularity for half-remembered dates). That machinery solves a
journaling problem Relay does not have.

### Bootstrap: relationships that predate the app

Existing relationships — a stepfather, a former landlord — were never captured
live. They enter through `Tag`, which asserts the relationship directly with no
event at all. Nothing is fabricated, and the graph is useful on day one
without inventing meetings that were never recorded.

An `EventProvenance` flag (live vs. recalled) is therefore **deferred**. It
would only be needed if users could type in past *meetings*, which is the
diary-shaped feature this section rules out. Note it cannot be retrofitted
onto existing rows, so it must be added at the same time as any such feature,
never after.

## Owner (who "I" am)

`Tag`, `Explain`, `Search`, and `Suggest` all have an implied first argument —
the person whose graph this is. The UI never asks "colleague of *whom*", and
"everyone I met in Seoul" filters by *someone's* participation. That anchor is
`Owner`.

```go
type Owner struct {
    PersonId string
}
```

**Why a singleton, not `Person.IsSelf bool`.** A flag spreads the invariant
"exactly one owner" across every row in the table, where nothing enforces it.
A bad import yields zero owners, a merge yields two, and every traversal
silently produces wrong answers. One record pointing at one person makes the
invariant structural.

**The owner is an ordinary `Person`.** Their name, birth date, and eventual
attributes reuse the same machinery as everyone else's. A separate profile
type would mean maintaining two parallel versions of all of it.

Rules the owner node carries that others do not:

- cannot be deleted
- cannot be merged into another Person
- never an endpoint of a suggested edge to itself

### The graph is general, not egocentric

Tempting shortcut: root everything at the user and drop `FromId`, storing only
`ToId`. The spec rules this out in its own example — AI suggesting
**"Father's friend"** requires a `FRIEND` edge between the user's father and a
third person, an edge the user is not an endpoint of.

So edges connect any two people, the owner is one node among many, and
`FromId` stays.

Consequences:

- **Perspective is rendering, not storage.** Store an edge in one canonical
  direction and render the label from the viewer's side using `Direction()`
  and `Inverse()`. Storing edges "from my perspective" makes third-party
  relationships inexpressible.
- **"Was the owner present?" is a real question.** For MVP every event is
  probably one the user attended, but a general graph permits recording events
  heard about secondhand. Whether `Owner` must appear in
  `Event.ParticipantIds` is undecided.
- **Identity across devices and imports** needs a stable anchor.
  `Person.PubKey` is the obvious candidate.

## Direct tagging

When the user tags a relationship directly, the result is **deterministic**:
`Status` is `StatusConfirmed`, `Confidence` is 1, and no inference runs. This
is the `Tag` verb in `service`.

Whether the tie also holds in reverse is a separate question, decided by the
relationship type rather than by tagging — see below.

Tagging is also the answer to person ambiguity. Rather than fuzzy-matching a
typed name against existing people and guessing, the user points at the person
they mean. Deferring NFC removed the `PubKey` handshake that used to make
identity automatic, and tagging replaces it without reintroducing a guess.

Consequence: `Tag` and `Confirm` are the only paths to a confirmed edge, and
neither is reachable from the AI path.

### Direction is a property of the type

A tag is not automatically bidirectional. Each `RelationshipType` declares how
it reads from the other side, via `RelationshipType.Direction()`:

- `DirectionSymmetric` — reverse carries the same label. `FRIEND`, `COLLEAGUE`.
- `DirectionInverse` — reverse carries a different label. `TEACHER`/`STUDENT`.
- `DirectionOneWay` — no reverse edge. The user remembers someone who does not
  know them.

Types missing from the `directions` table default to one-way, so adding a type
without deciding its reverse cannot silently fabricate an edge.

Still open: **whether the reverse is stored or derived.**

1. **Store one edge, derive the reverse on read**, using `Inverse()`. One row
   per tie, so confirm and reject stay single operations and the two
   directions cannot disagree.
2. **Store both edges.** Simpler reads, but the pair must be kept in sync on
   every confirm, reject, and end-date write, and can drift.

Leaning toward (1) — a tie is one fact, and storing it twice invites the two
copies to disagree.

`FAMILY` is currently marked symmetric, which is a placeholder and wrong in
general: the reverse of *father* is *son*, not *family*. It only holds because
`FAMILY` is too coarse to carry a role. Fixing it is the same work as the
role-and-direction gap under intrinsic qualifiers below.

## Relationship qualifiers ("prefixes")

`RelationshipType` is currently a flat, coarse enum (`FRIEND`, `FAMILY`,
`TEACHER`, `COLLEAGUE`). Real labels people use carry qualifiers that the enum
cannot express:

- "Stepfather" — a family tie, but qualified
- "The landlord who moved out" — a tie that *used to* hold

These are two different problems and should not be solved the same way.

### A. State qualifiers — model as Events

"Former", "ex-", "who moved out", "who left the company". These describe *when*
a relationship held, not what kind it is.

**Decision: these do not become enum values.** No `FORMER_LANDLORD`,
no `EX_COLLEAGUE`.

Reasons:

- The enum would multiply by every state — one variant per type per state.
- The end of a relationship is itself a dated fact with evidence. That is
  already what `Event` is. "Moved out" is an occurrence in the record.
- Relay's premise is explaining how a tie *began*. Explaining how it *ended*
  should reuse the same evidence machinery rather than invent a parallel one.

Implied shape when this is implemented — `Relationship` gains temporal validity
sourced from events:

```go
ValidFrom    time.Time
ValidUntil   time.Time // zero means still current
StartEventId string
EndEventId   string
```

"Former landlord" is then *derived at render time* from
`Type == LANDLORD && !ValidUntil.IsZero()`, not stored as a label.

### B. Intrinsic qualifiers — unresolved

"Step-", "half-", "adoptive-", "-in-law", "grand-". These describe the *nature*
of the tie. They are static, and unlike state qualifiers they cannot be
recovered from event history — a stepfather stays a stepfather regardless of
what happens afterward.

Three options, none chosen yet:

1. **Qualifier field** — `Relationship.Qualifier` (`step`, `half`, `adoptive`).
   Cheap; composes with any type.
2. **Finer-grained types** — `FAMILY_STEPFATHER`. Explicit and type-safe, but
   the enum explodes.
3. **Decomposed role** — `Type: FAMILY` + `Role: father` + `Qualifier: step`.
   Most flexible, most machinery.

### Related gap: role and direction

`FAMILY` does not say *father*, *sibling*, or *cousin*, and `Relationship` is
directed (`FromId` → `ToId`) while the label is not. One edge reads two ways:
A is the father of B, B is the son of A. Whatever solves intrinsic qualifiers
has to answer this too, so decide them together.

### Storage vs. display

"Stepfather" and "The landlord who moved out" are *renderings*, not storage.
Keep the stored form structured and generate the natural-language phrase from
it. Storing the phrase would put the same fact in two places and make it
unsearchable as a graph.

**Status: documented only. No code written for any of the above.**
