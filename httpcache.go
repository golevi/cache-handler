package httpcache

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/config"
	"github.com/golevi/cache-handler/handlers"
	"github.com/golevi/cache-handler/stores"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	cfg *config.Config
)

func init() {
	caddy.RegisterModule(Cache{})
	httpcaddyfile.RegisterGlobalOption("cache", parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective("cache", parseCaddyfileHandlerDirective)
}

// Cache stuff
type Cache struct {
	Config   config.Config
	Store    stores.CacheStore
	Deciders []func(c config.Config, w http.ResponseWriter, r *http.Request) bool

	logger *zap.Logger
}

type cacheResponse struct {
	Status        string      `json:"status"`
	StatusCode    int         `json:"status_code"`
	Headers       http.Header `json:"headers"`
	Body          []byte      `json:"body"`
	ContentLength int64       `json:"content_length"`
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

func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	labels := prometheus.Labels{"handler": "cache"}

	// Key
	key := key(r)

	// Loop through deciders to see whether or not this request should be cached
	// or if we should bypass it and send it to the origin.
	for _, decider := range c.Deciders {
		if decider(c.Config, w, r) {
			w.Header().Add("Cache-Status", "bypass")
			ch := httpMetrics.cacheBypass.With(labels)
			ch.Inc()

			return next.ServeHTTP(w, r)
		}
	}

	// If it is cached, we want to return it.
	if c.Store.Has(key) {
		w.Header().Add("Cache-Status", "hit")

		ch := httpMetrics.cacheHit.With(labels)
		ch.Inc()

		response, err := c.Store.Get(key)
		if err != nil {
			return err
		}

		var cr = cacheResponse{}
		err = json.Unmarshal((response).([]byte), &cr)
		if err != nil {
			return err
		}

		for name, values := range cr.Headers {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}

		w.WriteHeader(cr.StatusCode)
		w.Write(cr.Body)

		return nil
	}

	// Wasn't cached :(
	w.Header().Add("Cache-Status", "miss")

	ch := httpMetrics.cacheMiss.With(labels)
	ch.Inc()

	// Save to cache
	recorder := httptest.NewRecorder()

	// Next, please.
	next.ServeHTTP(recorder, r)

	body, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		c.logger.Error(err.Error())
	}

	cr := &cacheResponse{
		Status:        recorder.Result().Status,
		StatusCode:    recorder.Result().StatusCode,
		Headers:       recorder.Result().Header,
		Body:          body,
		ContentLength: recorder.Result().ContentLength,
	}

	response, err := json.Marshal(cr)
	if err != nil {
		c.logger.Error(err.Error())
	}

	c.Store.Put(key, response, time.Second*time.Duration(c.Config.Expire))

	for name, values := range cr.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(recorder.Code)
	w.Write(body)

	return nil
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

	c.Deciders = append(c.Deciders, handlers.URI)
	c.Deciders = append(c.Deciders, handlers.Method)
	c.Deciders = append(c.Deciders, handlers.Cookie)

	return nil
}

func key(r *http.Request) string {
	return "request:" + r.Method + ":" + r.Host + ":" + r.URL.Path
}

var (
	_ caddy.Provisioner           = (*Cache)(nil)
	_ caddyhttp.MiddlewareHandler = (*Cache)(nil)
	_ caddyfile.Unmarshaler       = (*Cache)(nil)
)
