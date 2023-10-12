# omg.httpServer

[![Go](https://github.com/dedalqq/omg.httpserver/actions/workflows/go.yml/badge.svg)](https://github.com/dedalqq/omg.httpserver/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/dedalqq/omg.httpServer/badge.svg?branch=master)](https://coveralls.io/github/dedalqq/omg.httpServer?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/dedalqq/omg.httpserver.svg)](https://pkg.go.dev/github.com/dedalqq/omg.httpserver)

omg.httpServer is the simple http handler for a standard http server. Also, its library contains the router for manage http requests. This library is best for creating API based on http request.

## Example

```go
package main

import (
	"context"
	"net/http"

	httpserver "github.com/dedalqq/omg.httpserver"
)

type Container struct{}

type AuthInfo struct{}

type UserRequest struct {
	UserID string `args:"user-id"`
}

type UserResponse struct{}

func main() {
	r := httpserver.NewRouter[*Container, *AuthInfo]()

	sr := r.SubRoute("/api/v1")

	sr.Add("/user/{user-id}", httpserver.Handler[*Container, *AuthInfo]{
		Get: httpserver.Create(func(ctx context.Context, c *Container, a *AuthInfo, rq *UserRequest) (*UserResponse, error) {
			return &UserResponse{}, nil
		}),
	})

	r.AddSwaggerSubRoute("/swagger.json", sr, httpserver.SwaggerOpt{})

	h := httpserver.NewHttpHandler(r, httpserver.Options{})

	h.SetAuthFunc(func(request *http.Request) (*AuthInfo, error) {
		return &AuthInfo{}, nil
	})

	server := &http.Server{
		Addr:    ":80",
		Handler: h,
	}

	server.ListenAndServe()
}

```
