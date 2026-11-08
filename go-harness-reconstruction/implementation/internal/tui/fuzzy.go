package tui

import (
	"sort"
	"strings"
	"unicode"
)

// fuzzyScore matches query letters in order within text (case-insensitive),
// like OpenCode's pickers. It favors exact substrings, then matches at word
// starts, then tight runs. ok is false when the letters do not all appear.
func fuzzyScore(query, text string) (score int, ok bool) {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(q) == 0 {
		return 0, true
	}
	t := []rune(strings.ToLower(text))
	if i := strings.Index(string(t), string(q)); i >= 0 {
		return 1000 - min(i, 500), true
	}
	qi, last := 0, -2
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if q[qi] == ' ' {
			qi++
			if qi == len(q) {
				break
			}
		}
		if t[ti] != q[qi] {
			continue
		}
		switch {
		case ti == last+1:
			score += 5 // consecutive letters
		case ti == 0 || !unicode.IsLetter(t[ti-1]) && !unicode.IsDigit(t[ti-1]):
			score += 8 // start of a word
		default:
			score++
		}
		last = ti
		qi++
	}
	return score, qi == len(q)
}

// rankFuzzy returns the indexes of items matching query, best first; ties
// keep their original order.
func rankFuzzy(query string, texts []string) []int {
	type hit struct{ index, score int }
	var hits []hit
	for i, text := range texts {
		if s, ok := fuzzyScore(query, text); ok {
			hits = append(hits, hit{i, s})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.index
	}
	return out
}
