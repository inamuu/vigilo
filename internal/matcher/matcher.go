package matcher

import "regexp"

type Matcher struct {
	patterns []compiledPattern
}

type compiledPattern struct {
	raw    string
	regexp *regexp.Regexp
}

func Compile(patterns []string) (*Matcher, error) {
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		compiled = append(compiled, compiledPattern{
			raw:    pattern,
			regexp: re,
		})
	}

	return &Matcher{patterns: compiled}, nil
}

func (m *Matcher) Match(line string) (string, bool) {
	for _, pattern := range m.patterns {
		if pattern.regexp.MatchString(line) {
			return pattern.raw, true
		}
	}

	return "", false
}
