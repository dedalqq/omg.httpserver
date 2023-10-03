package httpserver

import (
	"context"
	"fmt"
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

type HandlerFunc[C, A any] func(context.Context, C, A, *http.Request, []string) interface{}
type HandlerMiddleware[C, A any] func(HandlerFunc[C, A]) HandlerFunc[C, A]

type StdHandler func(context.Context, http.ResponseWriter, *http.Request, []string) bool

type apiDescription struct {
}

type MethodHandler[C, A any] struct {
	handlerFunc HandlerFunc[C, A]
	description apiDescription
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

type ResponseWithBody interface {
	Body() interface{}
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

type Error struct {
	cause     error
	HttpCode  int    `json:"code"`
	ErrorText string `json:"error"`
}

func NewError(code int, format string, a ...interface{}) Error {
	return Error{
		HttpCode:  code,
		ErrorText: fmt.Sprintf(format, a...),
	}
}

func Wrapf(err error, code int, format string, a ...interface{}) Error {
	return Error{
		cause:     err,
		HttpCode:  code,
		ErrorText: fmt.Sprintf(format, a...),
	}
}

func (e Error) Cause() error  { return e.cause }
func (e Error) Unwrap() error { return e.cause }

func (e Error) Error() string {
	causeErrText := ""
	if e.cause != nil {
		causeErrText = fmt.Sprintf(": %s", e.cause.Error())
	}

	return fmt.Sprintf("http error [%d] %s%s", e.HttpCode, e.ErrorText, causeErrText)
}

func (e Error) Code() int {
	return e.HttpCode
}
