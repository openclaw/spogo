package spotify

import "testing"

func TestParseResourceErrors(t *testing.T) {
	if _, err := ParseResource(""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := ParseResource("spotify:badtype:123"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := ParseResource("https://open.spotify.com/"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := ParseResource("spotify:track"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTypedIDNoExpectedType(t *testing.T) {
	res, err := ParseTypedID("spotify:track:t1", "")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Type != "track" || res.ID != "t1" {
		t.Fatalf("unexpected: %#v", res)
	}
}

func TestParseResourceShareURLs(t *testing.T) {
	for _, input := range []string{
		"https://open.spotify.com/intl-de/track/abc?si=share",
		"open.spotify.com/intl-pt/track/abc",
		"https://OPEN.SPOTIFY.COM/track/abc",
		"HTTPS://open.spotify.com/track/abc",
		"hTtP://open.spotify.com/track/abc",
		"SPOTIFY:track:abc",
		"https://open.spotify.com/embed/track/abc",
	} {
		res, err := ParseResource(input)
		if err != nil || res.URI != "spotify:track:abc" {
			t.Errorf("ParseResource(%q) = %+v, %v", input, res, err)
		}
	}
}

func TestParseResourceRejectsMalformedResources(t *testing.T) {
	for _, input := range []string{
		"spotify:track:", "spotify:track:abc:extra",
		"https://open.spotify.com/track/", "https://open.spotify.com/track/abc/extra",
		"https://example.test/track/abc?ref=open.spotify.com/",
		"https://user@open.spotify.com/track/abc",
	} {
		if _, err := ParseResource(input); err == nil {
			t.Errorf("accepted malformed resource %q", input)
		}
	}
}
