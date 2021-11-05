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
func NewServer(ctx context.Context, addr string, r Router, opt Options) *http.Server {
	log := opt.Logger

	if log == nil {
		log = &emptyLogger{}
	}

	return &http.Server{
		Addr: addr,
		Handler: &httpHandler{
			ctx: ctx,
			middlewares: []requestMiddleware{
				newLogMiddleware(log),
				newPanicHandlerMiddleware(log),
			},
			router: r,
			log:    log,
			gzip:   opt.SupportGZIP,
		},
	}
}
