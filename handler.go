package httpServer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func handleHttpRequest(ctx context.Context, router Router, r *http.Request) interface{} {
	ep, args := router.get(r.URL.Path)
	if ep == nil {
		return NewError(http.StatusNotFound, "Method not exist")
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
	case http.MethodDelete:
		handlerFunc = ep.Delete
	}

	if handlerFunc == nil {
		return NewError(http.StatusNotFound, "Method not exist")
	}

	for _, m := range ep.Middlewares {
		handlerFunc = m(handlerFunc)
	}

	return handlerFunc(ctx, r, args)
}

type handler struct {
	ctx         context.Context
	middlewares []requestMiddleware
	router      Router
	log         Logger
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := handleHttpRequest

	for _, m := range h.middlewares {
		handler = m(handler)
	}

	result := handler(h.ctx, h.router, r)

	err := r.Body.Close()
	if err != nil {
		h.log.Error(err)
	}

	if r, ok := result.(ResponseWithCode); ok {
		w.WriteHeader(r.Code())
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if r, ok := result.(ResponseWithContentType); ok {
		w.Header().Set("Content-Type", r.ContentType())
	} else {
		switch result.(type) {
		case io.Reader:
		default:
			w.Header().Set("Content-Type", "application/json")
		}
	}

	switch r := result.(type) {
	case io.Reader:
		_, err = io.Copy(w, r)
	default:
		err = json.NewEncoder(w).Encode(r)
	}

	if err != nil {
		h.log.Error(err)
	}
}
