package pattern

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type mode uint

type SyntaxError struct {
	msg string
	err error
}

func (e SyntaxError) Error() string { return e.msg }

func (e SyntaxError) Unwrap() error { return e.err }

const (
	Shortest     mode = 1 << iota // prefer the shortest match.
	Filenames                     // "*" and "?" don't match slashes; only "**" does
	Braces                        // support "{a,b}" and "{1..4}"
	EntireString                  // match the entire string using ^$ delimiters
	NoGlobCase                    // Do case-insensitive match (that is, use (?i) in the regexp)
)

var numRange = regexp.MustCompile(`^([+-]?\d+)\.\.([+-]?\d+)}`)

func Regexp(pat string, mode mode) (string, error) {
	needsEscaping := false
noopLoop:
	for _, r := range pat {
		switch r {
		// including those that need escaping since they are
		// regular expression metacharacters
		case '*', '?', '[', '\\', '.', '+', '(', ')', '|',
			']', '{', '}', '^', '$':
			needsEscaping = true
			break noopLoop
		}
	}
	if !needsEscaping && mode&EntireString == 0 { // short-cut without a string copy
		return pat, nil
	}
	closingBraces := []int{}
	var buf bytes.Buffer
	// Enable matching `\n` with the `.` metacharacter as globs match `\n`
	buf.WriteString("(?s)")
	dotMeta := false
	if mode&NoGlobCase != 0 {
		buf.WriteString("(?i)")
	}
	if mode&EntireString != 0 {
		buf.WriteString("^")
	}
writeLoop:
	for i := 0; i < len(pat); i++ {
		switch c := pat[i]; c {
		case '*':
			if mode&Filenames != 0 {
				if i++; i < len(pat) && pat[i] == '*' {
					if i++; i < len(pat) && pat[i] == '/' {
						buf.WriteString("(.*/|)")
						dotMeta = true
					} else {
						buf.WriteString(".*")
						dotMeta = true
						i--
					}
				} else {
					buf.WriteString("[^/]*")
					i--
				}
			} else {
				buf.WriteString(".*")
				dotMeta = true
			}
			if mode&Shortest != 0 {
				buf.WriteByte('?')
			}
		case '?':
			if mode&Filenames != 0 {
				buf.WriteString("[^/]")
			} else {
				buf.WriteByte('.')
				dotMeta = true
			}
		case '\\':
			if i++; i >= len(pat) {
				return "", &SyntaxError{msg: `\ at end of pattern`}
			}
			buf.WriteString(regexp.QuoteMeta(string(pat[i])))
		case '[':
			name, err := charClass(pat[i:])
			if err != nil {
				return "", &SyntaxError{msg: "charClass invalid", err: err}
			}
			if name != "" {
				buf.WriteString(name)
				i += len(name) - 1
				break
			}
			if mode&Filenames != 0 {
				for _, c := range pat[i:] {
					if c == ']' {
						break
					} else if c == '/' {
						buf.WriteString("\\[")
						continue writeLoop
					}
				}
			}
			buf.WriteByte(c)
			if i++; i >= len(pat) {
				return "", &SyntaxError{msg: "[ was not matched with a closing ]"}
			}
			switch c = pat[i]; c {
			case '!', '^':
				buf.WriteByte('^')
				if i++; i >= len(pat) {
					return "", &SyntaxError{msg: "[ was not matched with a closing ]"}
				}
			}
			if c = pat[i]; c == ']' {
				buf.WriteByte(']')
				if i++; i >= len(pat) {
					return "", &SyntaxError{msg: "[ was not matched with a closing ]"}
				}
			}
			rangeStart := byte(0)
		loopBracket:
			for ; i < len(pat); i++ {
				c = pat[i]
				buf.WriteByte(c)
				switch c {
				case '\\':
					if i++; i < len(pat) {
						buf.WriteByte(pat[i])
					}
					continue
				case ']':
					break loopBracket
				}
				if rangeStart != 0 && rangeStart > c {
					return "", &SyntaxError{msg: fmt.Sprintf("invalid range: %c-%c", rangeStart, c)}
				}
				if c == '-' {
					rangeStart = pat[i-1]
				} else {
					rangeStart = 0
				}
			}
			if i >= len(pat) {
				return "", &SyntaxError{msg: "[ was not matched with a closing ]"}
			}
		case '{':
			if mode&Braces == 0 {
				buf.WriteString(regexp.QuoteMeta(string(c)))
				break
			}
			innerLevel := 1
			commas := false
		peekBrace:
			for j := i + 1; j < len(pat); j++ {
				switch c := pat[j]; c {
				case '{':
					innerLevel++
				case ',':
					commas = true
				case '\\':
					j++
				case '}':
					if innerLevel--; innerLevel > 0 {
						continue
					}
					if !commas {
						break peekBrace
					}
					closingBraces = append(closingBraces, j)
					buf.WriteString("(?:")
					continue writeLoop
				}
			}
			if match := numRange.FindStringSubmatch(pat[i+1:]); len(match) == 3 {
				start, err1 := strconv.Atoi(match[1])
				end, err2 := strconv.Atoi(match[2])
				if err1 != nil || err2 != nil || start > end {
					return "", &SyntaxError{msg: fmt.Sprintf("invalid range: %q", match[0])}
				}
				// TODO: can we do better here?
				buf.WriteString("(?:")
				for n := start; n <= end; n++ {
					if n > start {
						buf.WriteByte('|')
					}
					fmt.Fprintf(&buf, "%d", n)
				}
				buf.WriteByte(')')
				i += len(match[0])
				break
			}
			buf.WriteString(regexp.QuoteMeta(string(c)))
		case ',':
			if len(closingBraces) == 0 {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			} else {
				buf.WriteByte('|')
			}
		case '}':
			if len(closingBraces) > 0 && closingBraces[len(closingBraces)-1] == i {
				buf.WriteByte(')')
				closingBraces = closingBraces[:len(closingBraces)-1]
			} else {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			}
		default:
			if c > 128 {
				buf.WriteByte(c)
			} else {
				buf.WriteString(regexp.QuoteMeta(string(c)))
			}
		}
	}
	if mode&EntireString != 0 {
		buf.WriteString("$")
	}
	// No `.` metacharacters were used, so don't return the (?s) flag.
	if !dotMeta {
		return string(buf.Bytes()[4:]), nil
	}
	return buf.String(), nil
}

func charClass(s string) (string, error) {
	if strings.HasPrefix(s, "[[.") || strings.HasPrefix(s, "[[=") {
		return "", fmt.Errorf("collating features not available")
	}
	name, ok := strings.CutPrefix(s, "[[:")
	if !ok {
		return "", nil
	}
	name, _, ok = strings.Cut(name, ":]]")
	if !ok {
		return "", fmt.Errorf("[[: was not matched with a closing :]]")
	}
	switch name {
	case "alnum", "alpha", "ascii", "blank", "cntrl", "digit", "graph",
		"lower", "print", "punct", "space", "upper", "word", "xdigit":
	default:
		return "", fmt.Errorf("invalid character class: %q", name)
	}
	return s[:len(name)+6], nil
}
