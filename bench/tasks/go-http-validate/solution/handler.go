package task

import (
	"encoding/json"
	"net/http"
)

// User is a stored user.
type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// UsersHandler creates users.
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if u.Name == "" || u.Age < 0 || u.Age > 150 {
		writeJSON(w, 400, map[string]string{"error": "invalid user"})
		return
	}
	writeJSON(w, http.StatusCreated, u)
}
