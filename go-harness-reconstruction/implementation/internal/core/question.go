package core

// Questions are runtime input, not authorization. Answers never grant tool access.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}
type Question struct {
	Question string           `json:"question"`
	Header   string           `json:"header"`
	Options  []QuestionOption `json:"options"`
	Multiple bool             `json:"multiple,omitempty"`
}
type QuestionRequest struct {
	Questions []Question `json:"questions"`
}
type QuestionReply struct {
	Answers [][]string `json:"answers"`
}
