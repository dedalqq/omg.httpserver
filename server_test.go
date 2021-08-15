package httpServer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
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
	r  io.Reader
	w  io.Writer
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

func TestServer(t *testing.T) {

	router := NewRouter()

	router.Add("/omg", Handler{
		Get: func(ctx context.Context, r *http.Request, args []string) interface{} {
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())

	server := NewServer(ctx, ":80", router, nil)

	l, client := newListenerAndClient()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := server.Serve(l)
		if err != nil && err.Error() != "http: Server closed" {
			t.Fatalf("Error: [%v]", err)
		}
	}()

	r, _ := http.NewRequest("GET", "http://localhost/omg", nil)

	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("Error: [%v]", err)
	}

	fmt.Println(resp.Status)

	cancel()
	err = server.Close()
	if err != nil {
		t.Fatalf("Error: [%v]", err)
	}

	wg.Wait()
}
