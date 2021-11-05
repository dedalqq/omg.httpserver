package httpserver

import (
	"context"
	"net/http"
)

func NewServer(ctx context.Context, addr string, r Router, gzip bool, log Logger) *http.Server {
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
			gzip:   gzip,
		},
	}
}
