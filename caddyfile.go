package httpcache

import (
	"regexp"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/golevi/cache-handler/config"
	"github.com/golevi/cache-handler/stores"
	"github.com/golevi/cache-handler/validators"
)

var (
	cfg *config.Config
)

func init() {
	caddy.RegisterModule(Cache{})
	httpcaddyfile.RegisterGlobalOption("cache", parseCaddyfileGlobalOption)
	httpcaddyfile.RegisterHandlerDirective("cache", parseCaddyfileHandlerDirective)
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
//		expire 120                              # Cache expiration in seconds
//		bypass {
//			paths wp-admin wp-login.php system  # WordPress and ExpressionEngine
//			methods post                        # Don't typically cache POST
//			cookies wordpress_logged_in_.*      # WordPress
//			# cookie exp_sessionid              # ExpressionEngine
//		}
//	}
//
// This may change.
func (c *Cache) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "bypass":
				for nesting := d.Nesting(); d.NextBlock(nesting); {
					switch d.Val() {
					case "paths":
						c.Config.Bypass.Paths = d.RemainingArgs()
					case "methods":
						c.Config.Bypass.Methods = d.RemainingArgs()
					case "cookies":
						c.Config.Bypass.Cookies = d.RemainingArgs()
					}
				}
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
	// Make sure regular expressions are valid
	for _, cc := range c.Config.Bypass.Cookies {
		c.Config.CookieRegexp = append(c.Config.CookieRegexp, regexp.MustCompile(cc))
	}

	c.logger = ctx.Logger(c)
	switch c.Config.Type {
	case "redis":
		c.Store = stores.NewRedisStore(c.Config.Host)
	case "file":
		c.Store = stores.NewFileStore()
	}

	c.Validators = append(c.Validators, validators.URI)
	c.Validators = append(c.Validators, validators.ShouldBypassHTTPMethod)
	c.Validators = append(c.Validators, validators.Cookie)

	return nil
}

var (
	_ caddy.Provisioner           = (*Cache)(nil)
	_ caddyhttp.MiddlewareHandler = (*Cache)(nil)
	_ caddyfile.Unmarshaler       = (*Cache)(nil)
)
