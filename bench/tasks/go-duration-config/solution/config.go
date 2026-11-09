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

// ParseConfig reads lines like "name=api" and "timeout=30" or "1m30s".
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
			if n, err := strconv.Atoi(v); err == nil {
				c.Timeout = time.Duration(n) * time.Second
				continue
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return c, err
			}
			c.Timeout = d
		}
	}
	return c, nil
}
