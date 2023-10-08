package httpserver

import (
	"net/http"
)

// Options for omg http handlers
type Options struct {
	SupportGZIP bool
	Logger      Logger
}

// NewServer creates and return new http server which the contains omg http handler
func NewServer[C, A any](addr string, r Router[C, A], opt Options, middlewares ...RequestMiddleware[C, A]) *http.Server {
	log := opt.Logger

	if log == nil {
		log = &emptyLogger{}
	}

	return &http.Server{
		Addr: addr,
		Handler: &HttpHandler[C, A]{
			middlewares: middlewares,
			router:      r,
			log:         log,
			gzip:        opt.SupportGZIP,
		},
	}
}
