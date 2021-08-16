package httpServer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
)

type addr struct {
	network string
	addr    string
}

func (a *addr) Network() string {
	return a.network
}

func (a *addr) String() string {
	return a.addr
}

type conn struct {
	wg *sync.WaitGroup

	r io.Reader
	w io.Writer
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *conn) Close() error {
	c.wg.Done()

	return nil
}

func (c *conn) LocalAddr() net.Addr {
	return &addr{}
}

func (c *conn) RemoteAddr() net.Addr {
	return &addr{}
}

func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

type listener struct {
	conn chan *conn
}

type transport struct {
	conn chan *conn
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	var wg sync.WaitGroup

	rawRequest := bytes.NewBuffer([]byte{})
	rawResponse := bytes.NewBuffer([]byte{})

	err := r.Write(rawRequest)
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	t.conn <- &conn{
		wg: &wg,
		w:  rawResponse,
		r:  rawRequest,
	}

	wg.Wait()

	return http.ReadResponse(bufio.NewReader(rawResponse), r)
}

func newListenerAndClient() (*listener, *http.Client) {
	ch := make(chan *conn)

	return &listener{
			conn: ch,
		}, &http.Client{
			Transport: &transport{
				conn: ch,
			},
		}
}

func (l *listener) Accept() (net.Conn, error) {
	conn := <-l.conn
	if conn != nil {
		return conn, nil
	}

	return nil, fmt.Errorf("listener is closed")
}

func (l *listener) Close() error {
	close(l.conn)

	return nil
}

func (l *listener) Addr() net.Addr {
	return nil
}

type serverRunnerFunc func(*http.Server)

func testRunner(t *testing.T, f func(context.Context, serverRunnerFunc, *http.Client) error) {
	ctx, cancel := context.WithCancel(context.Background())

	l, client := newListenerAndClient()

	var (
		err    error
		wg     sync.WaitGroup
		server *http.Server
	)

	serverRunner := func(s *http.Server) {
		server = s

		wg.Add(1)
		go func() {
			defer wg.Done()

			err = server.Serve(l)
		}()
	}

	e := f(ctx, serverRunner, client)
	if e != nil {
		t.Fatalf("Error: [%v]", err)
	}

	cancel()
	err = server.Close()
	if err != nil {
		t.Fatalf("Error: [%v]", err)
	}

	wg.Wait()

	if err != nil && err.Error() != "http: Server closed" {
		t.Fatalf("Error: [%v]", err)
	}
}

func TestServer(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		router.Add("/test", Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if resp.Status != "418 I'm a teapot" {
			t.Fail()
		}

		return nil
	})
}

func TestDefaultResponse(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		router.Add("/test", Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				return nil
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if resp.Status != "200 OK" {
			t.Fail()
		}

		return nil
	})
}

func TestDefaultHandler(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		router.Default(Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/some-path")
		if err != nil {
			return err
		}

		if resp.Status != "418 I'm a teapot" {
			t.Fail()
		}

		return nil
	})
}

func TestHandlerArgs(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		var arguments []string

		router.Add("/test/{any}", Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				arguments = args

				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		_, err := cl.Get("http://localhost/test/some-test-data")
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(arguments, []string{"some-test-data"}) {
			t.Fail()
		}

		return nil
	})
}

func TestNotFound(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if resp.Status != "404 Not Found" {
			t.Fail()
		}

		return nil
	})
}

func TestMethodNotSupported(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		router.Add("/test", Handler{
			Post: func(ctx context.Context, r *http.Request, args []string) interface{} {
				return nil
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if resp.Status != "404 Not Found" {
			t.Fail()
		}

		return nil
	})
}

func TestMiddleware(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		var testValue string

		router.Add("/test", Handler{
			Middlewares: []HandlerMiddleware{
				func(handler HandlerFunc) HandlerFunc {
					return func(ctx context.Context, r *http.Request, args []string) interface{} {
						ctx = context.WithValue(ctx, "test", "test")

						return handler(ctx, r, args)
					}
				},
			},

			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				testValue = ctx.Value("test").(string)

				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		_, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if testValue != "test" {
			t.Fail()
		}

		return nil
	})
}

func TestSubRoute(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		router.SubRoute("/test").Add("/sub-test", Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		resp, err := cl.Get("http://localhost/test/sub-test")
		if err != nil {
			return err
		}

		if resp.Status != "418 I'm a teapot" {
			t.Fail()
		}

		return nil
	})
}

func TestAllMethods(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter()

		var methods []string

		router.Add("/test", Handler{
			Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
				methods = append(methods, "GET")

				return NewError(http.StatusTeapot, "teapot")
			},

			Post: func(ctx context.Context, r *http.Request, args []string) interface{} {
				methods = append(methods, "POST")

				return NewError(http.StatusTeapot, "teapot")
			},

			Put: func(ctx context.Context, r *http.Request, args []string) interface{} {
				methods = append(methods, "PUT")

				return NewError(http.StatusTeapot, "teapot")
			},

			Patch: func(ctx context.Context, r *http.Request, args []string) interface{} {
				methods = append(methods, "PATCH")

				return NewError(http.StatusTeapot, "teapot")
			},

			Delete: func(ctx context.Context, r *http.Request, args []string) interface{} {
				methods = append(methods, "DELETE")

				return NewError(http.StatusTeapot, "teapot")
			},
		})

		run(NewServer(ctx, ":80", router, nil))

		doRequest := func(method string, url string) {
			req, err := http.NewRequest(method, url, nil)
			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err.Error())
			}

			if resp.Status != "418 I'm a teapot" {
				t.Fail()
			}
		}

		doRequest(http.MethodGet, "http://localhost/test")
		doRequest(http.MethodPost, "http://localhost/test")
		doRequest(http.MethodPut, "http://localhost/test")
		doRequest(http.MethodPatch, "http://localhost/test")
		doRequest(http.MethodDelete, "http://localhost/test")

		if !reflect.DeepEqual(methods, []string{"GET", "POST", "PUT", "PATCH", "DELETE"}) {
			t.Fail()
		}

		return nil
	})
}
