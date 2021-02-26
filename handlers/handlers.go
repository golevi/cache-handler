package handlers

import (
	"net/http"

	"github.com/golevi/cache-handler/config"
)

// Decider is an interface for deciding how specific requests should be handled.
type Decider interface {
	ShouldBypass(c config.Config, w http.ResponseWriter, r *http.Request) bool
}
