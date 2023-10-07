package httpserver

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func handleHttpRequest[C, A any](ctx context.Context, router Router[C, A], c C, af AuthFunc[A], w http.ResponseWriter, r *http.Request) (interface{}, bool) {
	ep, argsPlace, args := router.get(r.URL.Path)
	if ep == nil {
		return NewError(http.StatusNotFound, "method not exist"), true
	}

	if ep.StdHandler != nil {
		ctn := ep.StdHandler(ctx, w, r, args)
		if !ctn {
			return nil, false
		}
	}

	var handler *MethodHandler[C, A]

	switch r.Method {
	case http.MethodGet:
		if ep.Get != nil {
			handler = ep.Get
		}
	case http.MethodPost:
		if ep.Post != nil {
			handler = ep.Post
		}
	case http.MethodPut:
		if ep.Put != nil {
			handler = ep.Put
		}
	case http.MethodPatch:
		if ep.Patch != nil {
			handler = ep.Patch
		}
	case http.MethodDelete:
		if ep.Delete != nil {
			handler = ep.Delete
		}
	}

	if handler == nil {
		return NewError(http.StatusNotFound, "method not supported"), true
	}

	handlerFunc := handler.handlerFunc

	for _, m := range ep.Middlewares {
		handlerFunc = m(handlerFunc)
	}

	// if handler.description.authRequired && af == nil {
	// 	return NewError(http.StatusInternalServerError, "method not supported"), true
	// }

	var (
		authInfo A
		err      error
	)

	if af != nil {
		authInfo, err = af(r)
		if err != nil && handler.description.authRequired {
			return err, true
		}
	}

	return handlerFunc(ctx, c, authInfo, r, argsPlace, args), true
}

type httpHandler[C, A any] struct {
	ctx         context.Context
	middlewares []RequestMiddleware[C, A]
	container   C
	authFunc    AuthFunc[A]
	router      Router[C, A]
	log         Logger
	gzip        bool
}

// ServeHTTP is a Handler responds to an HTTP request.
func (h *httpHandler[C, A]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := handleHttpRequest[C, A]

	for _, m := range h.middlewares {
		handler = m(handler)
	}

	result, ctn := handler(h.ctx, h.router, h.container, h.authFunc, w, r)
	if !ctn {
		return
	}

	err := r.Body.Close()
	if err != nil {
		h.log.Error(err)
		return
	}

	gzipAccept := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

	if gzipAccept && h.gzip {
		w.Header().Set("Content-Encoding", "gzip")
	}

	switch r := result.(type) {
	case ResponseWithContentType:
		w.Header().Set("Content-Type", r.ContentType())
	case io.Reader:
	default:
		w.Header().Set("Content-Type", "application/json")
	}

	if rs, ok := result.(ResponseWithCookie); ok {
		for _, c := range rs.Cookie() {
			http.SetCookie(w, c)
		}
	}

	switch r := result.(type) {
	case ResponseWithCode:
		w.WriteHeader(r.Code())
	case error:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}

	body := result
	if r, ok := result.(ResponseWithBody); ok {
		body = r.Body()
	}

	if gzipAccept && h.gzip {
		gw := gzip.NewWriter(w)
		err = writeBody(gw, body)
		if err != nil {
			h.log.Error(err)
			return
		}

		err = gw.Close()
		if err != nil {
			h.log.Error(err)
		}

		return
	}

	err = writeBody(w, body)
	if err != nil {
		h.log.Error(err)
	}
}

func writeBody(w io.Writer, body interface{}) error {
	var err error

	switch r := body.(type) {
	case io.Reader:
		_, err = io.Copy(w, r)
	default:
		err = json.NewEncoder(w).Encode(r)
	}

	return err
}
