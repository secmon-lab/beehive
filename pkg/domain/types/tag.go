package types

import (
	"regexp"

	"github.com/m-mizutani/goerr/v2"
)

var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_]*[a-zA-Z0-9]$`)

// Tag represents a source tag
type Tag string

// NewTag creates a validated tag
func NewTag(s string) (Tag, error) {
	if len(s) == 0 {
		return "", goerr.New("tag cannot be empty")
	}
	if len(s) == 1 {
		// Single character tag must be alphanumeric
		if !regexp.MustCompile(`^[a-zA-Z0-9]$`).MatchString(s) {
			return "", goerr.New("single character tag must be alphanumeric", goerr.V("tag", s))
		}
		return Tag(s), nil
	}
	if len(s) > 63 {
		return "", goerr.New("tag too long (max 63 chars)", goerr.V("tag", s), goerr.V("length", len(s)))
	}
	if !tagPattern.MatchString(s) {
		return "", goerr.New("tag must match pattern [a-zA-Z0-9][a-zA-Z0-9-_]*[a-zA-Z0-9]", goerr.V("tag", s))
	}
	return Tag(s), nil
}

func (t Tag) String() string {
	return string(t)
}

// Tags represents a list of tags
type Tags []Tag

// NewTags creates validated tags from strings
func NewTags(ss []string) (Tags, error) {
	tags := make(Tags, 0, len(ss))
	for _, s := range ss {
		tag, err := NewTag(s)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// Strings converts tags to string slice
func (ts Tags) Strings() []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = t.String()
	}
	return result
}
