package validators_test

import (
	"net/http/httptest"
	"testing"

	"github.com/golevi/cache-handler/config"
	"github.com/golevi/cache-handler/validators"
)

func TestMethod(t *testing.T) {
	cfg := config.Config{
		Bypass: config.Bypass{
			Methods: []string{"post", "head"},
		},
	}
	req := httptest.NewRequest("get", "/", nil)
	res := httptest.NewRecorder()

	if validators.ShouldBypassHTTPMethod(cfg, res, req) {
		t.Error("nope")
	}
}
