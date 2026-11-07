package session

import (
	"encoding/json"

	"example.com/independent-harness/internal/core"
)

type UserTurn struct {
	Number int
	Prompt string
}

func isUserRecord(record Record) bool {
	if record.Type != "message" {
		return false
	}
	var role struct{ Role string }
	return json.Unmarshal(record.Data, &role) == nil && role.Role == "user"
}

func userRecordContent(record Record) string {
	var message core.Message
	_ = json.Unmarshal(record.Data, &message)
	return message.Content
}

// RecentUserTurns lists durable prompt boundaries newest first, even if
// compaction replaced those messages in the model's context.
func (s *Store) RecentUserTurns(limit int) []UserTurn {
	if limit <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	indices := make([]int, min(limit, len(s.records)))
	count := 0
	for i, record := range s.records {
		if isUserRecord(record) {
			indices[count%len(indices)] = i
			count++
		}
	}
	turns := make([]UserTurn, 0, min(count, len(indices)))
	for i := count - 1; i >= max(0, count-len(indices)); i-- {
		turns = append(turns, UserTurn{
			Number: i + 1,
			Prompt: userRecordContent(s.records[indices[i%len(indices)]]),
		})
	}
	return turns
}
