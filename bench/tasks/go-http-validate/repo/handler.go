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

// UsersHandler creates users.
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	json.NewDecoder(r.Body).Decode(&u)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}
