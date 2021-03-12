# Caddy Cache-Handler

[![Go](https://github.com/golevi/cache-handler/actions/workflows/go.yml/badge.svg)](https://github.com/golevi/cache-handler/actions/workflows/go.yml)

* [Config](config)
* [Handlers](handlers)
* [Stores](stores)

## Build

```bash
export GOOS=linux
export GOARCH=amd64
xcaddy build --with github.com/golevi/cache-handler
tar -czf caddy.tar.gz caddy
```

## Contributing

Good reads:

* [Caching Tutorial](https://www.mnot.net/cache_docs/) by [Mark Nottingham](https://www.mnot.net/)
* [HTTP caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching) by [Mozilla Developer Network](https://developer.mozilla.org/en-US/)
* [Prevent unnecessary network requests with the HTTP Cache](https://web.dev/http-cache/) by [web.dev](https://web.dev/)
