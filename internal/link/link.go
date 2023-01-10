package link

// makeRoutePattern builds a pattern matching URL for the mux.
func MakeRoutePattern(prefix *string) string {
	return MakeBaseURL(prefix) + "/{id:.*}"
}

// makeHumanPattern builds a human-friendly URL for display.
func MakeHumanPattern(prefix *string) string {
	return MakeBaseURL(prefix) + "/{id}"
}

// makeBaseURL creates the base URL before any mux pattern matching.
func MakeBaseURL(prefix *string) string {
	if prefix == nil || *prefix == "" {
		return ""
	}

	return "/" + *prefix
}
