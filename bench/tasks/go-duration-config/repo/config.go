package task

import (
	"strconv"
	"strings"
	"time"
)

// Config holds service settings parsed from key=value lines.
type Config struct {
	Name    string
	Timeout time.Duration
}

// ParseConfig reads lines like "name=api" and "timeout=30" (seconds).
func ParseConfig(text string) (Config, error) {
	c := Config{Timeout: 30 * time.Second}
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "name":
			c.Name = v
		case "timeout":
			n, err := strconv.Atoi(v)
			if err != nil {
				return c, err
			}
			c.Timeout = time.Duration(n) * time.Second
		}
	}
	return c, nil
}
