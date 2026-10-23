package loflv

import (
	"strings"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/domain"
)

// participantIndex is a lookup structure for matching by normalized name.
type participantIndex struct {
	byName map[string]domain.Participant
}

// newParticipantIndex builds an index from participants that have SIDs.
func newParticipantIndex(participants []domain.Participant) *participantIndex {
	idx := &participantIndex{
		byName: make(map[string]domain.Participant),
	}
	for _, p := range participants {
		if p.SID == "" {
			continue
		}
		key := nameKey(p.Surname, p.Name)
		idx.byName[key] = p
	}
	return idx
}

// Match finds a participant matching the given first/last name.
func (idx *participantIndex) Match(firstName, lastName string) (domain.Participant, bool) {
	key := nameKey(lastName, firstName)
	p, ok := idx.byName[key]
	return p, ok
}

func nameKey(surname, firstName string) string {
	return strings.ToLower(strings.TrimSpace(surname)) + "\t" + strings.ToLower(strings.TrimSpace(firstName))
}
