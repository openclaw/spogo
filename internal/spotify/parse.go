package spotify

import (
	"errors"
	"net/url"
	"strings"
)

var ErrUnsupportedType = errors.New("unsupported spotify type")

var supportedTypes = map[string]struct{}{
	"track":    {},
	"album":    {},
	"artist":   {},
	"playlist": {},
	"show":     {},
	"episode":  {},
}

type Resource struct {
	Type string
	ID   string
	URI  string
}

func ParseResource(input string) (Resource, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Resource{}, errors.New("empty input")
	}
	if strings.HasPrefix(strings.ToLower(input), "spotify:") {
		parts := strings.Split(input, ":")
		if len(parts) != 3 {
			return Resource{}, errors.New("invalid spotify uri")
		}
		return typedResource(parts[1], parts[2])
	}
	if strings.HasPrefix(strings.ToLower(input), "open.spotify.com/") {
		input = "https://" + input
	}
	if strings.Contains(input, "://") {
		parsed, err := url.Parse(input)
		if err != nil {
			return Resource{}, err
		}
		if (parsed.Scheme != "https" && parsed.Scheme != "http") ||
			!strings.EqualFold(parsed.Hostname(), "open.spotify.com") || parsed.User != nil {
			return Resource{}, errors.New("invalid spotify url")
		}
		segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(segments) == 3 && (strings.HasPrefix(segments[0], "intl-") || segments[0] == "embed") {
			segments = segments[1:]
		}
		if len(segments) != 2 {
			return Resource{}, errors.New("invalid spotify url")
		}
		return typedResource(segments[0], segments[1])
	}
	return Resource{ID: input}, nil
}

func typedResource(kind, id string) (Resource, error) {
	if !isSupportedType(kind) {
		return Resource{}, ErrUnsupportedType
	}
	if strings.TrimSpace(id) == "" {
		return Resource{}, errors.New("spotify id required")
	}
	return Resource{Type: kind, ID: id, URI: "spotify:" + kind + ":" + id}, nil
}

func ParseTypedID(input, expectedType string) (Resource, error) {
	res, err := ParseResource(input)
	if err != nil {
		return Resource{}, err
	}
	if expectedType == "" {
		return res, nil
	}
	if res.Type == "" {
		res.Type = expectedType
		res.URI = "spotify:" + expectedType + ":" + res.ID
		return res, nil
	}
	if res.Type != expectedType {
		return Resource{}, errors.New("unexpected spotify type")
	}
	return res, nil
}

func isSupportedType(kind string) bool {
	_, ok := supportedTypes[kind]
	return ok
}

func isContextURI(uri string) bool {
	return strings.Contains(uri, ":album:") || strings.Contains(uri, ":playlist:") || strings.Contains(uri, ":show:")
}
