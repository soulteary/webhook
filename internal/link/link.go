package link

// makeRoutePattern builds a pattern matching URL for the chi router.
// Returns the base pattern without wildcard - we'll register both /{id} and /{id}/* routes.
func MakeRoutePattern(prefix *string) string {
	return MakeBaseURL(prefix) + "/{id}"
}

// makeHumanPattern builds a human-friendly URL for display.
func MakeHumanPattern(prefix *string) string {
	return MakeBaseURL(prefix) + "/{id}"
}

// makeBaseURL creates the base URL before any router pattern matching.
func MakeBaseURL(prefix *string) string {
	if prefix == nil || *prefix == "" {
		return ""
	}

	return "/" + *prefix
}
