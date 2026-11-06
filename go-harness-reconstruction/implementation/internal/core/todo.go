package core

// Todo is ordered session task state. Status and Priority are deliberately plain
// strings: the pinned reference does not enforce enums or transition rules.
type Todo struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}
