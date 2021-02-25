package httpcache

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/stores/filestore"
	"github.com/golevi/cache-handler/stores/redisstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	Expire int      `json:"expire"`
	Cookie []string `json:"cookie"`
}

// Cache stuff
type Cache struct {
	Config

	Store CacheStore

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

	// Might eventually do something with regex, but for now, only check if the
	// beginning of the URI matches any of the bypass strings.
	uri := r.RequestURI[1:]
	segments := strings.Split(uri, "/")
	if len(segments) > 0 {
		uri = segments[0]
	}

	// Check the config to see if this URI should NOT be cached
	if contains(c.Config.Bypass, uri) {
		w.Header().Add("Cache-Status", "bypass")

		ch := httpMetrics.cacheBypass.With(labels)
		ch.Inc()
		return next.ServeHTTP(w, r)
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
// 		expire 120                              # Cache expiration in seconds
// 		method post                             # Don't typically cache POST
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
		c.Store = redisstore.NewRedisStore(c.Host)
	case "file":
		c.Store = filestore.NewFileStore()
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

func key(r *http.Request) string {
	return "request:" + r.Method + ":" + r.Host + ":" + r.URL.Path
}

var (
	_ caddy.Provisioner           = (*Cache)(nil)
	_ caddyhttp.MiddlewareHandler = (*Cache)(nil)
	_ caddyfile.Unmarshaler       = (*Cache)(nil)
)
