package validators

import (
	"net/http"
	"regexp"

	"github.com/golevi/cache-handler/config"
)

// Cookie compares the config cookie values against the ones sent by the request
// and if one matches, we want to bypass the cache.
func Cookie(c config.Config, w http.ResponseWriter, r *http.Request) bool {
	// Loop through all the cookies from the request
	for _, cookie := range r.Cookies() {
		// Loop through all the regexp cookie names from the config
		for _, cc := range c.Cookie {
			// Compile the regexp
			rge := regexp.MustCompile(cc)
			// If we find a cookie, return true that we should bypass the cache.
			if rge.Find([]byte(cookie.Name)) != nil {
				return true
			}
		}
	}

	return false
}
