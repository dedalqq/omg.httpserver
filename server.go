package httpserver

import (
	"context"
	"net/http"
)

func NewServer(ctx context.Context, addr string, r Router, log Logger) *http.Server {
	if log == nil {
		log = &emptyLogger{}
	}

	return &http.Server{
		Addr: addr,
		Handler: &handler{
			ctx: ctx,
			middlewares: []requestMiddleware{
				newLogMiddleware(log),
				newPanicHandlerMiddleware(log),
			},
			router: r,
			log:    log,
		},
	}
}
