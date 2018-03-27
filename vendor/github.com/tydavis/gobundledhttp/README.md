# gobundledhttp

Provides convenience functions to generate an *http.Client with bundled CA certificates.

Available Methods:

```go
// Normal SSL-capable client
sslclient := gobundledhttp.NewClient()
// Disables SSL certificate checking
nosslclient := gobundledhttp.InsecureClient()

// CtxBundled returns an oauth2 context with bundled http client
context := gobundledhttp.CtxBundled()
// Get the default cert pool to build your own client
myX509pool := gobundledhttp.GetPool()
```

To update certificates (needed only if the cacerts source changes):

```bash
go generate
```
