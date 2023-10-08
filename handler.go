package httpserver

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
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

type HttpHandler[C, A any] struct {
	middlewares []RequestMiddleware[C, A]
	container   C
	authFunc    AuthFunc[A]
	router      Router[C, A]
	log         Logger
	gzip        bool
}

func NewHttpHandler[C, A any](r Router[C, A], opt Options, middlewares ...RequestMiddleware[C, A]) *HttpHandler[C, A] {
	log := opt.Logger

	if log == nil {
		log = &emptyLogger{}
	}

	return &HttpHandler[C, A]{
		middlewares: middlewares,
		router:      r,
		log:         log,
		gzip:        opt.SupportGZIP,
	}
}

func (h *HttpHandler[C, A]) SetContainer(container C) *HttpHandler[C, A] {
	h.container = container
	return h
}

// ServeHTTP is a Handler responds to an HTTP request.
func (h *HttpHandler[C, A]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := handleHttpRequest[C, A]

	for _, m := range h.middlewares {
		handler = m(handler)
	}

	result, ctn := handler(r.Context(), h.router, h.container, h.authFunc, w, r)
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

	//if _, ok := result.(error); !ok && contentType == "" {
	//	rt := reflect.TypeOf(result)
	//	for rt.Kind() == reflect.Pointer {
	//		rt = rt.Elem()
	//	}
	//
	//	if rt.Kind() == reflect.Struct {
	//		contentType = "application/json"
	//	}
	//}

	if r, ok := result.(ResponseWithContentType); ok {
		w.Header().Set("Content-Type", r.ContentType())
	} else if rr := reflect.TypeOf(r); rr != nil {
		for rr.Kind() == reflect.Pointer {
			rr = rr.Elem()
		}

		if rr.Kind() == reflect.Struct {
			w.Header().Set("Content-Type", "application/json")
		}
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

	if gzipAccept && h.gzip {
		gw := gzip.NewWriter(w)
		err = writeBody(gw, result)
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

	err = writeBody(w, result)
	if err != nil {
		h.log.Error(err)
	}
}

func writeBody(w io.Writer, body interface{}) error {
	var err error

	switch r := body.(type) {
	case []byte:
		_, err = w.Write(r)
	case string:
		_, err = fmt.Fprint(w, r)
	case io.Reader:
		_, err = io.Copy(w, r)
	default:
		err = json.NewEncoder(w).Encode(r)
	}

	return err
}
