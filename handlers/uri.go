package handlers

import (
	"net/http"
	"strings"

	"github.com/golevi/cache-handler/config"
)

// URI takes the config Bypass and checks those URIs against the current request
// uri to see if it matches. If it does, we return true because we should bypass
// this requeust.
func URI(c config.Config, w http.ResponseWriter, r *http.Request) bool {
	uri := r.RequestURI[1:]
	segments := strings.Split(uri, "/")
	if len(segments) > 0 {
		uri = segments[0]
	}

	return contains(c.Bypass, uri)
}
