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

type requestHandler func(ctx context.Context, router Router, r *http.Request) interface{}
type requestMiddleware func(requestHandler) requestHandler

type HandlerFunc func(context.Context, *http.Request, []string) interface{}
type HandlerMiddleware func(HandlerFunc) HandlerFunc

type Handler struct {
	Middlewares []HandlerMiddleware

	Get    HandlerFunc
	Post   HandlerFunc
	Put    HandlerFunc
	Delete HandlerFunc
	Patch  HandlerFunc
}

type ResponseWithCode interface {
	Code() int
}

type ResponseWithContentType interface {
	ContentType() string
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

type Response struct {
	Body interface{}

	Code        int
	ContentType string
	Cookie      []*http.Cookie
}

func NewResponse(body interface{}) *Response {
	return &Response{
		Body: body,
	}
}

func (r *Response) SetCode(code int) *Response {
	r.Code = code

	return r
}

func (r *Response) SetContentType(contentType string) *Response {
	r.ContentType = contentType

	return r
}

func (r *Response) AddCookie(c *http.Cookie) *Response {
	r.Cookie = append(r.Cookie, c)

	return r
}
