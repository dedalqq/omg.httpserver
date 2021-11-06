package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

func newLogMiddleware(log Logger) requestMiddleware {
	return func(h requestHandler) requestHandler {
		return func(ctx context.Context, router Router, w http.ResponseWriter, r *http.Request) (interface{}, bool) {
			start := time.Now()
			result, ctn := h(ctx, router, w, r)
			duration := time.Since(start)

			code := http.StatusOK
			if r, ok := result.(ResponseWithCode); ok {
				code = r.Code()
			}

			log.Info("[%s] %s [Code:%d] %s", r.Method, r.URL.Path, code, duration.String())

			if err, ok := result.(error); ok {
				log.Error(err) // TODO print only one
			}

			return result, ctn
		}
	}
}

func newPanicHandlerMiddleware(log Logger) requestMiddleware {
	return func(h requestHandler) requestHandler {
		return func(ctx context.Context, router Router, w http.ResponseWriter, r *http.Request) (res interface{}, ctn bool) {
			defer func() {
				r := recover()

				if r == nil {
					return
				}

				switch e := r.(type) {
				case error:
					log.Error(errors.Wrapf(e, "Panic"))
				default:
					log.Error(errors.Errorf("Panic: %v", e))
				}

				res = NewError(http.StatusInternalServerError, "internal error")
				ctn = true
			}()

			res, ctn = h(ctx, router, w, r)

			return
		}
	}
}
