package httpcache

import (
	"strconv"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/config"
	"github.com/golevi/cache-handler/stores"
	"github.com/golevi/cache-handler/validators"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cfg *config.Config
)

func init() {
	caddy.RegisterModule(Cache{})
	httpcaddyfile.RegisterGlobalOption("cache", parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective("cache", parseCaddyfileHandlerDirective)
}

var httpMetrics = struct {
	init        sync.Once
	cacheHit    *prometheus.CounterVec
	cacheMiss   *prometheus.CounterVec
	cacheBypass *prometheus.CounterVec
}{
	init: sync.Once{},
}

// CaddyModule returns the Caddy module information.
func (Cache) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.cache",
		New: func() caddy.Module { return new(Cache) },
	}
}

func parseCaddyfileGlobalOption(d *caddyfile.Dispenser) (interface{}, error) {
	cfg = &config.Config{}
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

// UnmarshalCaddyfile parses plugin settings from Caddyfile.
//
//	{
//		order cache first
//		cache {
// 			type <redis>|<file>
//			host localhost:6379
// 		}
//	}
//
//	cache {
// 		expire 120                              # Cache expiration in seconds
// 		method post                             # HTTP Methods you want to bypass
// 		bypass wp-admin wp-login.php system     # WordPress and ExpressionEngine
// 		# cookie exp_sessionid                  # ExpressionEngine
// 		cookie wordpress_logged_in_.*           # WordPress
//	}
//
// This may change.
func (c *Cache) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "bypass":
				c.Config.Bypass = d.RemainingArgs()
			case "cookie":
				c.Config.Cookie = d.RemainingArgs()
			case "expire":
				expire, _ := strconv.Atoi(d.RemainingArgs()[0])
				c.Config.Expire = expire
			case "method":
				c.Config.Method = d.RemainingArgs()
			}
		}
	}

	return nil
}

// Provision _
func (c *Cache) Provision(ctx caddy.Context) error {
	const ns, sub = "caddy", "http"

	basicLabels := []string{"handler"}
	httpMetrics.cacheHit = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: sub,
		Name:      "requests_cache_hit",
		Help:      "Counter of HTTP cache hit requests",
	}, basicLabels)
	httpMetrics.cacheMiss = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: sub,
		Name:      "requests_cache_miss",
		Help:      "Counter of HTTP cache miss requests",
	}, basicLabels)
	httpMetrics.cacheBypass = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: ns,
		Subsystem: sub,
		Name:      "requests_cache_bypass",
		Help:      "Counter of HTTP cache bypass requests",
	}, basicLabels)

	c.logger = ctx.Logger(c)
	switch c.Config.Type {
	case "redis":
		c.Store = stores.NewRedisStore(c.Config.Host)
	case "file":
		c.Store = stores.NewFileStore()
	}

	c.Deciders = append(c.Deciders, validators.URI)
	c.Deciders = append(c.Deciders, validators.Method)
	c.Deciders = append(c.Deciders, validators.Cookie)

	return nil
}

var (
	_ caddy.Provisioner           = (*Cache)(nil)
	_ caddyhttp.MiddlewareHandler = (*Cache)(nil)
	_ caddyfile.Unmarshaler       = (*Cache)(nil)
)
