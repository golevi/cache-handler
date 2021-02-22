package httpcache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
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
	httpcaddyfile.RegisterGlobalOption("cache", parseCaddyfileGlobalOption)
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
	Type string `json:"type,omitempty"`
	Host string `json:"host,omitempty"`

	Bypass []string `json:"bypass"`
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

	// Check the config to see if this URI should NOT be cached
	if contains(c.Config.Bypass, r.RequestURI[1:]) {
		w.Header().Add("Cache-Status", "bypass")
		return next.ServeHTTP(w, r)
	}

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
func parseCaddyfileGlobalOption(d *caddyfile.Dispenser) (interface{}, error) {
	cfg = &Config{}
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "host":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				cfg.Host = d.Val()
			case "type":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				cfg.Type = d.Val()
			}
		}
	}

	return nil, nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	c := &Cache{}
	if cfg != nil {
		c.Config = *cfg
	}

	err := c.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
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
	switch c.Config.Type {
	case "redis":
		c.Store = redisstore.NewRedisStore(c.Host)
	}

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
	_ caddyfile.Unmarshaler       = (*Cache)(nil)
)
