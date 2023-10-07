package httpserver

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
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

type TestContainer struct {
	data string
}

type TestUserData struct {
	userID int
}

type handler = Handler[*TestContainer, *TestUserData]

type TestRequest struct {
}

type TestResponse struct {
	Data string `json:"data,omitempty"`
}

func TestServer(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		testHandler := func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
			return nil, NewError(http.StatusTeapot, "teapot")
		}

		router.Add("/test", handler{
			Get: Create(testHandler, AuthRequired()),
		})

		run(NewServer(ctx, ":80", router, Options{}))

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
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Add("/test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				return nil, nil
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

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
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Default(handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				return nil, NewError(http.StatusTeapot, "teapot")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

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

type TestUserDataWithArguments struct {
	Args1 string `args:"args1"`
	Args2 int    `args:"args2"`
}

func TestHandlerAnyArgs(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		var argument string

		router.Add("/first-test/{args1}", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				t.Fail()

				return nil, nil
			}),
		})

		router.Add("/first-test/{args1}/second-test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestUserDataWithArguments) (*TestResponse, error) {
				argument = r.Args1

				return nil, NewError(http.StatusTeapot, "teapot")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

		_, err := cl.Get("http://localhost/first-test/some-test-data/second-test")
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(argument, "some-test-data") {
			t.Fail()
		}

		return nil
	})
}

func TestHandlerIntArgs(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		var argument int

		router.Add("/first-test/{args2}", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				t.Fail()

				return nil, nil
			}),
		})

		router.Add("/first-test/{args2}/second-test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestUserDataWithArguments) (*TestResponse, error) {
				argument = r.Args2

				return nil, NewError(http.StatusTeapot, "teapot")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

		_, err := cl.Get("http://localhost/first-test/123/second-test")
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(argument, 123) {
			t.Fail()
		}

		return nil
	})
}

func TestNotFound(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		run(NewServer(ctx, ":80", router, Options{}))

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

func TestPanic(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Add("/test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				panic("just panic")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}, NewPanicHandlerMiddleware[*TestContainer, *TestUserData](&emptyLogger{})))

		resp, err := cl.Get("http://localhost/test")
		if err != nil {
			return err
		}

		if resp.Status != "500 Internal Server Error" {
			t.Fail()
		}

		return nil
	})
}

func TestMethodNotSupported(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Add("/test", handler{
			Post: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				return nil, nil
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

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

//func TestMiddleware(t *testing.T) {
//	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
//		router := NewRouter[*TestContainer, *TestUserData]()
//
//		var testValue string
//
//		router.Add("/test", handler{
//			Middlewares: []HandlerMiddleware[*TestContainer, *TestUserData]{
//				func(handler HandlerFunc) HandlerFunc {
//					return func(ctx context.Context, r *http.Request, args []string) interface{} {
//						ctx = context.WithValue(ctx, "test", "test")
//
//						return handler(ctx, r, args)
//					}
//				},
//			},
//
//			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
//				testValue = ctx.Value("test").(string)
//
//				return nil, NewError(http.StatusTeapot, "teapot")
//			}),
//		})
//
//		run(NewServer(ctx, ":80", router, Options{}))
//
//		_, err := cl.Get("http://localhost/test")
//		if err != nil {
//			return err
//		}
//
//		if testValue != "test" {
//			t.Fail()
//		}
//
//		return nil
//	})
//}

func TestSubRoute(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		router.SubRoute("/test").Add("/sub-test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				return nil, NewError(http.StatusTeapot, "teapot")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

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
		router := NewRouter[*TestContainer, *TestUserData]()

		var methods []string

		router.Add("/test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				methods = append(methods, "GET")

				return nil, NewError(http.StatusTeapot, "teapot")
			}),

			Post: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				methods = append(methods, "POST")

				return nil, NewError(http.StatusTeapot, "teapot")
			}),

			Put: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				methods = append(methods, "PUT")

				return nil, NewError(http.StatusTeapot, "teapot")
			}),

			Patch: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				methods = append(methods, "PATCH")

				return nil, NewError(http.StatusTeapot, "teapot")
			}),

			Delete: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
				methods = append(methods, "DELETE")

				return nil, NewError(http.StatusTeapot, "teapot")
			}),
		})

		run(NewServer(ctx, ":80", router, Options{}))

		doRequest := func(method string, url string) {
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				t.Fatal(err.Error())
			}

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

//func TestSuccessResponse(t *testing.T) {
//	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
//		router := NewRouter[*TestContainer, *TestUserData]()
//
//		router.Add("/test", handler{
//			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*TestResponse, error) {
//				return &TestResponse{
//					Data: "test",
//				}, nil
//			}),
//		})
//
//		run(NewServer(ctx, ":80", router, Options{}))
//
//		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//
//		resp, err := cl.Do(req)
//		if err != nil {
//			t.Fatal(err.Error())
//		}
//
//		if resp.Status != "418 I'm a teapot" {
//			t.Fail()
//		}
//
//		if resp.Header.Get("Content-Type") != "application/test-type" {
//			t.Fail()
//		}
//
//		data, err := io.ReadAll(resp.Body)
//		if err != nil {
//			t.Fail()
//		}
//
//		if string(data) != "{\"data\":\"test\"}\n" {
//			t.Fail()
//		}
//
//		return nil
//	})
//}

func TestGZIP(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Add("/test", handler{
			Get: Create(func(ctx context.Context, c *TestContainer, u *TestUserData, r *TestRequest) (*strings.Reader, error) {
				return strings.NewReader("the some test text"), nil
			}),
		})

		run(NewServer(ctx, ":80", router, Options{SupportGZIP: true}))

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		if err != nil {
			t.Fatal(err.Error())
		}

		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err.Error())
		}

		if resp.Status != "200 OK" {
			t.Fail()
		}

		if resp.Header.Get("Content-Encoding") != "gzip" {
			t.Fail()
		}

		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			t.Fail()
		}

		data, err := io.ReadAll(r)
		if err != nil {
			t.Fail()
		}

		if string(data) != "the some test text" {
			fmt.Println(string(data))
			t.Fail()
		}

		return nil
	})
}

func TestStdHandler(t *testing.T) {
	testRunner(t, func(ctx context.Context, run serverRunnerFunc, cl *http.Client) error {
		router := NewRouter[*TestContainer, *TestUserData]()

		router.Add("/test", handler{
			StdHandler: func(ctx context.Context, w http.ResponseWriter, r *http.Request, args []string) bool {
				w.WriteHeader(418)

				return false
			},
		})

		run(NewServer(ctx, ":80", router, Options{}))

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
