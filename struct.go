package httpserver

import (
	"context"
	"net/http"
)

type Logger interface {
	Info(string, ...interface{})
	Error(error)
}

type emptyLogger struct{}

func (*emptyLogger) Info(string, ...interface{}) {}
func (*emptyLogger) Error(error)                 {}

type AuthFunc[A any] func(*http.Request) (A, error)

type RequestHandler[C, A any] func(context.Context, Router[C, A], C, AuthFunc[A], http.ResponseWriter, *http.Request) (interface{}, bool)
type RequestMiddleware[C, A any] func(RequestHandler[C, A]) RequestHandler[C, A]

type HandlerFunc[C, A any] func(context.Context, C, A, *http.Request, []string, []string) interface{}
type HandlerMiddleware[C, A any] func(HandlerFunc[C, A]) HandlerFunc[C, A]

type StdHandler func(context.Context, http.ResponseWriter, *http.Request, []string) bool

type objectType struct {
	name        string
	description string
	object      *apiType
}

type apiDescription struct {
	authRequired bool

	headers OrderedMap[apiType]
	args    OrderedMap[apiType]
	query   OrderedMap[apiType]

	requestObject objectType

	respContentType   string
	successStatusCode int
	responseObject    objectType
}

type MethodHandler[C, A any] struct {
	description apiDescription
	handlerFunc HandlerFunc[C, A]
}

type Handler[C, A any] struct {
	StdHandler StdHandler

	Middlewares []HandlerMiddleware[C, A]

	Get    *MethodHandler[C, A]
	Post   *MethodHandler[C, A]
	Put    *MethodHandler[C, A]
	Delete *MethodHandler[C, A]
	Patch  *MethodHandler[C, A]
}

type ResponseWithCode interface {
	Code() int
}

type ResponseWithContentType interface {
	ContentType() string
}

type ResponseWithCookie interface {
	Cookie() []*http.Cookie
}

type NoContent struct{}

func (n NoContent) Code() int { return http.StatusNoContent }
