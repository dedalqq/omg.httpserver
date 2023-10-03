package httpserver

import (
	"context"
	"net/http"
)

// Options for omg http handlers
type Options struct {
	SupportGZIP bool
	Logger      Logger
}

// NewServer creates and return new http server which the contains a omg http handler
func NewServer[C, A any](ctx context.Context, addr string, r Router[C, A], opt Options, middlewares ...RequestMiddleware[C, A]) *http.Server {
	log := opt.Logger

	if log == nil {
		log = &emptyLogger{}
	}

	return &http.Server{
		Addr: addr,
		Handler: &httpHandler[C, A]{
			ctx:         ctx,
			middlewares: middlewares,
			router:      r,
			log:         log,
			gzip:        opt.SupportGZIP,
		},
	}
}
