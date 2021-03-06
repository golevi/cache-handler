# Caddy Cache-Handler

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
