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

type requestHandler func(context.Context, Router, http.ResponseWriter, *http.Request) (interface{}, bool)
type requestMiddleware func(requestHandler) requestHandler

type HandlerFunc func(context.Context, *http.Request, []string) interface{}
type HandlerMiddleware func(HandlerFunc) HandlerFunc
type StdHandler func(context.Context, http.ResponseWriter, *http.Request, []string) bool

type Handler struct {
	StdHandler StdHandler

	Middlewares []HandlerMiddleware

	Get    HandlerFunc
	Post   HandlerFunc
	Put    HandlerFunc
	Delete HandlerFunc
	Patch  HandlerFunc
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

type Response struct {
	body interface{}

	code        int
	contentType string
	cookie      []*http.Cookie
}

func NewResponse(body interface{}) *Response {
	return &Response{
		body: body,
		code: http.StatusOK,
	}
}

func (r *Response) Body() interface{} {
	return r.body
}

func (r *Response) SetCode(code int) *Response {
	r.code = code

	return r
}

func (r *Response) Code() int {
	return r.code
}

func (r *Response) SetContentType(contentType string) *Response {
	r.contentType = contentType

	return r
}

func (r *Response) ContentType() string {
	return r.contentType
}

func (r *Response) AddCookie(c *http.Cookie) *Response {
	r.cookie = append(r.cookie, c)

	return r
}

func (r *Response) Cookie() []*http.Cookie {
	return r.cookie
}
