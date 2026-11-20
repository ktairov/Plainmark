// Package preview preserves Plainmark's Markdown conventions before rendering.
package preview

import (
	"regexp"
	"strings"
)

var (
	fencePattern    = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	amountPattern   = regexp.MustCompile(`^\$\d[\d,.]*`)
	numericVariable = regexp.MustCompile(`^[a-zA-Z]+\$`)
	parenPattern    = regexp.MustCompile(`^\(([^)\n]*)\)([^\n]*)`)
	mathOperators   = regexp.MustCompile(`[+*/=^_\\]`)
	breakPattern    = regexp.MustCompile(`(?i)^<br\s*/?\s*>`)
)

func escaped(s string, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

func currency(s string, i int) bool {
	if escaped(s, i) || (i > 0 && s[i-1] == '$') {
		return false
	}
	amount := amountPattern.FindString(s[i:])
	if amount == "" {
		return false
	}
	rest := s[i+len(amount):]
	if strings.HasPrefix(rest, "$") || numericVariable.MatchString(rest) {
		return false
	}
	next := strings.TrimLeft(rest, " \t\r\n")
	if strings.HasPrefix(next, "(") {
		m := parenPattern.FindStringSubmatch(next)
		if len(m) > 0 && strings.Contains(m[2], "$") && (strings.HasPrefix(strings.TrimSpace(m[2]), "$") || mathOperators.MatchString(m[1])) {
			return false
		}
	} else if len(next) > 0 && strings.ContainsRune("+-*/=^_{}\\[%", rune(next[0])) {
		return false
	}
	if len(next) > 0 && (next[0] == '<' || next[0] == '>') && !breakPattern.MatchString(next) {
		return false
	}
	return true
}

// Normalize converts recognized single-dollar math to parenthesized math so
// MyGo's conservative numeric-dollar heuristic does not reject $5x$ or prices
// inside a formula. Currency and fenced/inline code retain their meaning.
func Normalize(source string) string {
	var out, plain strings.Builder
	var marker byte
	length := 0
	flush := func() { out.WriteString(segment(plain.String())); plain.Reset() }
	for _, line := range strings.SplitAfter(source, "\n") {
		m := fencePattern.FindStringSubmatch(line)
		if marker == 0 && len(m) > 0 {
			flush()
			marker = m[1][0]
			length = len(m[1])
			out.WriteString(line)
		} else if marker != 0 {
			out.WriteString(line)
			s := strings.TrimSpace(line)
			if len(s) >= length && strings.Trim(s, string(marker)) == "" {
				marker = 0
			}
		} else {
			plain.WriteString(line)
		}
	}
	flush()
	return out.String()
}

func segment(s string) string {
	var out strings.Builder
	ticks := 0
	for i := 0; i < len(s); {
		if s[i] == '`' {
			j := i + 1
			for j < len(s) && s[j] == '`' {
				j++
			}
			n := j - i
			if ticks == 0 {
				ticks = n
			} else if ticks == n {
				ticks = 0
			}
			out.WriteString(s[i:j])
			i = j
			continue
		}
		if ticks == 0 && s[i] == '$' && !escaped(s, i) {
			if currency(s, i) {
				out.WriteString(`\$`)
				i++
				continue
			}
			width := 1
			for i+width < len(s) && s[i+width] == '$' {
				width++
			}
			end := -1
			for j := i + width; j < len(s); j++ {
				if s[j] != '$' || escaped(s, j) {
					continue
				}
				z := j + 1
				for z < len(s) && s[z] == '$' {
					z++
				}
				if z-j == width {
					end = j
					break
				}
				j = z - 1
			}
			if end >= 0 {
				if width == 1 {
					out.WriteString(`\(`)
					out.WriteString(s[i+1 : end])
					out.WriteString(`\)`)
				} else {
					out.WriteString(s[i : end+width])
				}
				i = end + width
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}
