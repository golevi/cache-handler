# Config Example

```
{
    order cache first

    cache {
        type redis
        host 127.0.0.1:6379                 # Redis IP/Port
    }
}

http://localhost:9090
cache {
    expire 120                              # Cache expiration in seconds
    method post                             # HTTP Methods you want to bypass
    bypass wp-admin wp-login.php system     # URIs - WordPress and ExpressionEngine
    # cookie uses regular expressions
    # cookie exp_sessionid                  # ExpressionEngine
    cookie wordpress_logged_in_.*           # WordPress
}
reverse_proxy localhost:8080
```
