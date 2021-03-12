package validators

import (
	"net/http"
	"strings"

	"github.com/golevi/cache-handler/config"
)

// ShouldBypassHTTPMethod _
func ShouldBypassHTTPMethod(c config.Config, w http.ResponseWriter, r *http.Request) bool {
	return contains(c.Bypass.Methods, strings.ToLower(r.Method))
}
