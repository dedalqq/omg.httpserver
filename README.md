# omg.httpServer

[![Go](https://github.com/dedalqq/omg.httpserver/actions/workflows/go.yml/badge.svg)](https://github.com/dedalqq/omg.httpserver/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/dedalqq/omg.httpServer/badge.svg?branch=master)](https://coveralls.io/github/dedalqq/omg.httpServer?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/dedalqq/omg.httpserver.svg)](https://pkg.go.dev/github.com/dedalqq/omg.httpserver)

omg.httpServer is the simple http handler for a standard http server. Also, its library contains the router for manage http requests. This library is best for creating API based  on http request.

## Example

```go
package main

import (
	"context"
	"net/http"

	httpserver "github.com/dedalqq/omg.httpserver"
)

func main() {
	router := httpserver.NewRouter()

	router.Add("/test", httpserver.Handler{
		Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
			return "hello world!"
		},
	})

	httpserver.NewServer(context.Background(), ":80", router, httpserver.Options{}).ListenAndServe()
}
```
