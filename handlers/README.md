# handlers

These functions decide if the request should be cached. They accept the parsed
[config][1], a [http.ResponseWriter][2], and a [http.Request][3].

There are handlers for the following:

* [Cookies][4]
* [HTTP Methods][5]
* [Page URIs][6]


[1]: https://github.com/golevi/cache-handler/blob/main/config
[2]: https://pkg.go.dev/net/http#ResponseWriter
[3]: https://pkg.go.dev/net/http#Request
[4]: cookie.go
[5]: method.go
[6]: uri.go
