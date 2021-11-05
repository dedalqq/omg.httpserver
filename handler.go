package httpserver

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func handleHttpRequest(ctx context.Context, router Router, r *http.Request) interface{} {
	ep, args := router.get(r.URL.Path)
	if ep == nil {
		return NewError(http.StatusNotFound, "method not exist")
	}

	var (
		handlerFunc HandlerFunc
	)

	switch r.Method {
	case http.MethodGet:
		handlerFunc = ep.Get
	case http.MethodPost:
		handlerFunc = ep.Post
	case http.MethodPut:
		handlerFunc = ep.Put
	case http.MethodPatch:
		handlerFunc = ep.Patch
	case http.MethodDelete:
		handlerFunc = ep.Delete
	}

	if handlerFunc == nil {
		return NewError(http.StatusNotFound, "method not supported")
	}

	for _, m := range ep.Middlewares {
		handlerFunc = m(handlerFunc)
	}

	return handlerFunc(ctx, r, args)
}

type httpHandler struct {
	ctx         context.Context
	middlewares []requestMiddleware
	router      Router
	log         Logger
	gzip        bool
}

// ServeHTTP is a Handler responds to an HTTP request.
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := handleHttpRequest

	for _, m := range h.middlewares {
		handler = m(handler)
	}

	result := handler(h.ctx, h.router, r)

	err := r.Body.Close()
	if err != nil {
		h.log.Error(err)
		return
	}

	gzipAccept := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

	if gzipAccept && h.gzip {
		w.Header().Set("Content-Encoding", "gzip")
	}

	if rs, ok := result.(ResponseWithContentType); ok {
		w.Header().Set("Content-Type", rs.ContentType())
	} else {
		switch result.(type) {
		case io.Reader:
		default:
			w.Header().Set("Content-Type", "application/json")
		}
	}

	if rs, ok := result.(ResponseWithCookie); ok {
		for _, c := range rs.Cookie() {
			http.SetCookie(w, c)
		}
	}

	if r, ok := result.(ResponseWithCode); ok {
		w.WriteHeader(r.Code())
	} else {
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
			panic("failed to write buffer")
		}

		err = gw.Close()
	} else {
		err = writeBody(w, body)
	}

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
