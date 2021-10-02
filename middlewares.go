package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

func newLogMiddleware(log Logger) requestMiddleware {
	return func(h requestHandler) requestHandler {
		return func(ctx context.Context, router Router, r *http.Request) interface{} {
			start := time.Now()
			result := h(ctx, router, r)
			duration := time.Since(start)

			code := http.StatusOK
			if r, ok := result.(ResponseWithCode); ok {
				code = r.Code()
			}

			log.Info("[%s] %s [Code:%d] %s", r.Method, r.URL.Path, code, duration.String())

			if err, ok := result.(error); ok {
				log.Error(err) // TODO print only one
			}

			return result
		}
	}
}

func newPanicHandlerMiddleware(log Logger) requestMiddleware {
	return func(h requestHandler) requestHandler {
		return func(ctx context.Context, router Router, r *http.Request) (res interface{}) {
			defer func() {
				r := recover()

				if r == nil {
					return
				}

				switch e := r.(type) {
				case error:
					log.Error(e)
				default:
					log.Error(errors.Errorf("Panic: %v", e))
				}

				res = NewError(http.StatusInternalServerError, "internal error")
			}()

			res = h(ctx, router, r)

			return
		}
	}
}
