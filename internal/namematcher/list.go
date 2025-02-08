package namematcher

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// List of file patterns used to ban unwanted files
// Pattern can be a part of the path, a file name..

type List struct {
	re       []*regexp.Regexp
	patterns []string
}

func New(patterns ...string) (List, error) {
	l := List{}
	for _, name := range patterns {
		err := l.Set(name)
		if err != nil {
			return List{}, err
		}
	}
	return l, nil
}

func MustList(patterns ...string) List {
	l, err := New(patterns...)
	if err != nil {
		panic(err.Error())
	}
	return l
}

func (l List) Match(name string) bool {
	for _, re := range l.re {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

func fetchRune(b []byte) ([]byte, rune) {
	r, size := utf8.DecodeRune(b)
	b = b[size:]
	return b, r
}

// transform a glob styled pattern into a regular expression
func patternToRe(pattern string) (*regexp.Regexp, error) {
	var r strings.Builder
	var inBrackets bool
	var b rune
	buf := []byte(pattern)

	r.WriteString("(?i)") // make the pattern case insensitive
	isFirstRune := true

	for len(buf) > 0 {
		buf, b = fetchRune(buf)
		switch b {
		case '/':
			if isFirstRune {
				r.WriteString(`(^|/)`)
			} else {
				r.WriteRune('/')
			}
		case '*':
			r.WriteString(`[^/]*`)
		case '?':
			r.WriteString(`[^/]`)
		case '.', '^', '$', '(', ')', '|':
			r.WriteRune('\\')
			r.WriteRune(b)
		case '\\':
			r.WriteRune(b)
			buf, b = fetchRune(buf)
			r.WriteRune(b)
		case '[':
			inBrackets = true
			r.WriteRune(b)
		brackets:
			for len(buf) > 0 {
				buf, b = fetchRune(buf)
				switch b {
				case ']':
					inBrackets = false
					r.WriteRune(b)
					break brackets
				default:
					lCase, uCase := unicode.ToLower(b), unicode.ToUpper(b)
					r.WriteRune(lCase)
					if lCase != uCase {
						r.WriteRune(uCase)
					}
				}
			}
		default:
			r.WriteRune(b)
		}
		isFirstRune = false
	}
	if inBrackets {
		return nil, fmt.Errorf("invalid file name pattern: %s", pattern)
	}
	re, err := regexp.Compile(r.String())
	if err != nil {
		return nil, fmt.Errorf("invalid file name pattern: %s", pattern)
	}
	return re, nil
}

/*
	Implements the flag.Value interface for the list of banned files
	Check the validity of the pattern
*/

func (l *List) Set(s string) error {
	if l == nil {
		return errors.New("namematcher  list not initialized")
	}
	if s == "" {
		return nil
	}
	re, err := patternToRe(s)
	if err != nil {
		return err
	}
	l.re = append(l.re, re)
	l.patterns = append(l.patterns, s)
	return nil
}

func (l List) String() string {
	var s strings.Builder
	for i, pattern := range l.patterns {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteRune('\'')
		s.WriteString(pattern)
		s.WriteRune('\'')
	}
	return s.String()
}

func (l *List) Get() any {
	return *l
}

func (l List) Type() string {
	return "FileList"
}
