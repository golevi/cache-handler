package handlers

import (
	"net/http"
	"strings"

	"github.com/golevi/cache-handler/config"
)

// Method checks the config methods we should cache against the request method
// and determines whether or not this request should be cached.
func Method(c config.Config, w http.ResponseWriter, r *http.Request) bool {
	return contains(c.Method, strings.ToLower(r.Method))
}
