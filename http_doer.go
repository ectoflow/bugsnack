package bugsnack

import "net/http"

// Doer is a helper interface used within bugsnack
// to avoid requiring an *http.Client be passed around
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
