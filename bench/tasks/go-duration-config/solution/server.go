package task

import "time"

// Server is a stub service.
type Server struct {
	Name        string
	ReadTimeout time.Duration
}

// NewServer builds a server from config.
func NewServer(c Config) Server {
	return Server{Name: c.Name, ReadTimeout: c.Timeout}
}
