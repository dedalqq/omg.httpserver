package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

type Request struct {
	Header1 string `header:"x-header-1"`

	ObjectID int `args:"object-id"`

	FooBar string `query:"foo-bar"`

	F1   string `json:"f1"`
	Int1 int    `json:"intValue" desc:"int value"`
}

type SubResponse struct {
	FF int
}

type Response struct {
	F2 string `json:"f2"`

	F3 struct {
		F4 string
	}

	FF6 SubResponse
	FF7 *SubResponse
}

func TestSwagger(t *testing.T) {
	r := NewRouter[any, any]()

	r.Add("/omg", Handler[any, any]{
		Get: Create(func(ctx context.Context, c any, a any, rq *Request) (*Response, error) {
			return nil, nil
		}),
		Patch: Create(func(ctx context.Context, c any, a any, rq *Request) (*io.Reader, error) {
			return nil, nil
		}),
	})

	swagger, err := r.renderSwagger(context.Background(), nil, nil, struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.MarshalIndent(swagger, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(data))
}
