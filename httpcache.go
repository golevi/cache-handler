package httpcache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/stores/redisstore"
	"go.uber.org/zap"
)

var (
	cfg *Config
)

func init() {
	caddy.RegisterModule(Cache{})
	httpcaddyfile.RegisterHandlerDirective("cache", parseCaddyfileHandlerDirective)
}

// CacheStore represents a way to cache
type CacheStore interface {
	Get(key string) (interface{}, error)
	Has(key string) bool
	Put(key string, value interface{}, expiration time.Duration)
}

// Config options
type Config struct {
	Host string `json:"host,omitempty"`
}

// Cache stuff
type Cache struct {
	Config

	Store CacheStore

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Cache) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.cache",
		New: func() caddy.Module { return new(Cache) },
	}
}

func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	key := strings.Join([]string{r.Host, r.RequestURI, r.Method}, "-")

	// If it is cached, we want to return it.
	if c.Store.Has(key) {
		w.Header().Add("Cache-Status", "hit")
		response, err := c.Store.Get(key)
		if err != nil {
			return err
		}

		w.Write(response.([]byte))

		return nil
	}

	// Wasn't cached :(
	w.Header().Add("Cache-Status", "miss")

	// Save to cache
	recorder := httptest.NewRecorder()

	// Next, please.
	next.ServeHTTP(recorder, r)

	w.WriteHeader(recorder.Code)
	content := recorder.Body.Bytes()

	c.Store.Put(key, content, time.Second*30)

	w.Write(content)

	return nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	c := &Cache{}
	if cfg != nil {
		c.Config = *cfg
	}

	return c, nil
}

// UnmarshalCaddyfile sets up the handler from Caddyfile tokens. Syntax:
//
//	cache {
//		bypass wp-admin wp-login system
//	}
//
// This may change.
func (c *Cache) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "bypass":
				c.Config.Bypass = d.RemainingArgs()
			}
		}
	}

	return nil
}

// Provision _
func (c *Cache) Provision(ctx caddy.Context) error {
	c.logger = ctx.Logger(c)
	c.Store = redisstore.NewRedisStore("")

	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

var (
	_ caddy.Provisioner           = (*Cache)(nil)
	_ caddyhttp.MiddlewareHandler = (*Cache)(nil)
)
