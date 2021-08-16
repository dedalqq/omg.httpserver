# omg.httpServer

[![Go](https://github.com/dedalqq/omg.httpServer/actions/workflows/go.yml/badge.svg)](https://github.com/dedalqq/omg.httpServer/actions/workflows/go.yml)

## Example

```go
package main

import (
	"context"
	"net/http"

	"github.com/dedalqq/omg.httpServer"
)

func main() {
	router := httpServer.NewRouter()

	router.Add("/test", httpServer.Handler{
		Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
			return "hello world!"
		},
	})

	httpServer.NewServer(context.Background(), ":80", router, nil).ListenAndServe()
}
```