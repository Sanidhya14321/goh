package session

import (
	"encoding/json"

	"github.com/Sanidhya14321/goh/internal/core"
)

// MessageRecord is a journaled message with the record it came from: its
// position in the journal and its time (zero for version 1 records, which
// carry none).
type MessageRecord struct {
	Index   int
	Time    int64
	Message core.Message
}

// Len is how many records the journal holds. Records are only appended
// while a store is open, so a caller can tell the journal changed by
// comparing lengths, without copying any record.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

// RecentMessages returns the latest message records, oldest first, as many
// as fit budget by weigh (at least one when any exists). It walks the
// journal backwards and decodes only the messages it returns, so the cost
// follows the selected tail rather than the session's length.
func (s *Store) RecentMessages(budget int, weigh func(core.Message) int) []MessageRecord {
	s.mu.Lock()
	// Records are append-only and never modified: a snapshot of the slice
	// header is a stable view after the lock is released.
	records := s.records[:len(s.records):len(s.records)]
	s.mu.Unlock()
	var out []MessageRecord
	used := 0
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Type != "message" {
			continue
		}
		var message core.Message
		if json.Unmarshal(records[i].Data, &message) != nil {
			continue
		}
		used += weigh(message)
		if used > budget && len(out) > 0 {
			break
		}
		out = append(out, MessageRecord{Index: i, Time: records[i].Time, Message: message})
	}
	for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
		out[l], out[r] = out[r], out[l]
	}
	return out
}

// Last returns a copy of the latest record whose type is one of types.
func (s *Store) Last(types ...string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.records) - 1; i >= 0; i-- {
		for _, t := range types {
			if s.records[i].Type == t {
				r := s.records[i]
				r.Data = append(json.RawMessage(nil), r.Data...)
				return r, true
			}
		}
	}
	return Record{}, false
}
