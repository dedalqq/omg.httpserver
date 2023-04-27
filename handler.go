package httpserver

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func handleHttpRequest(ctx context.Context, router Router, w http.ResponseWriter, r *http.Request) (interface{}, bool) {
	ep, args := router.get(r.URL.Path)
	if ep == nil {
		return NewError(http.StatusNotFound, "method not exist"), true
	}

	if ep.StdHandler != nil {
		ctn := ep.StdHandler(ctx, w, r, args)
		if !ctn {
			return nil, false
		}
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
		return NewError(http.StatusNotFound, "method not supported"), true
	}

	for _, m := range ep.Middlewares {
		handlerFunc = m(handlerFunc)
	}

	return handlerFunc(ctx, r, args), true
}

type httpHandler struct {
	ctx         context.Context
	middlewares []RequestMiddleware
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

	result, ctn := handler(h.ctx, h.router, w, r)
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
