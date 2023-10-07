package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Option func(*apiDescription)

func AuthRequired() Option {
	return func(d *apiDescription) {
		d.authRequired = true
	}
}

type fieldHandler struct {
	tag string
	fn  func(string, interface{}) error
}

func trim(tagValue string, value interface{}) error {
	switch v := value.(type) {
	case *string:
		if v != nil {
			*v = strings.TrimSpace(*v)
		}
	}

	return nil
}

func parseArgs(tagValue string, targetValue interface{}, argsPlace []string, args []string) error {
	var argValue string

	for i, name := range argsPlace {
		if name == tagValue {
			argValue = args[i]
		}
	}

	switch v := targetValue.(type) {
	case *string:
		*v = argValue
	case *int, *int8, *int16, *int32, *int64:
		intVal, err := strconv.Atoi(argValue)
		if err != nil {
			return fmt.Errorf("value [%s]=[%v] is not a number", tagValue, argValue)
		}

		reflect.ValueOf(targetValue).Elem().SetInt(int64(intVal))
	case *uint, *uint8, *uint16, *uint32, *uint64:
		intVal, err := strconv.Atoi(argValue)
		if err != nil {
			return fmt.Errorf("value [%s]=[%v] is not a number", tagValue, argValue)
		}

		reflect.ValueOf(targetValue).Elem().SetUint(uint64(intVal))
	default:
		panic("type not supported")
	}

	return nil
}

func parseQuery(tagValue string, targetValue interface{}, url *url.URL) error {
	queryValue := url.Query().Get(tagValue)
	if queryValue == "" {
		return nil
	}

	switch v := targetValue.(type) {
	case *string:
		*v = queryValue
	case *int, *int8, *int16, *int32, *int64:
		intVal, err := strconv.Atoi(queryValue)
		if err != nil {
			return fmt.Errorf("value [%s]=[%v] is not a number", tagValue, queryValue)
		}

		reflect.ValueOf(targetValue).Elem().SetInt(int64(intVal))
	case *uint, *uint8, *uint16, *uint32, *uint64:
		intVal, err := strconv.Atoi(queryValue)
		if err != nil {
			return fmt.Errorf("value [%s]=[%v] is not a number", tagValue, queryValue)
		}

		reflect.ValueOf(targetValue).Elem().SetUint(uint64(intVal))
	case *bool:
		switch queryValue {
		case "true", "True", "1":
			*v = true
		case "false", "False", "0":
			*v = false
		default:
			return fmt.Errorf("incorrect bool value [%s]=[%v]", tagValue, queryValue)
		}
	default:
		return json.Unmarshal([]byte(tagValue), targetValue)
	}

	return nil
}

func handleStructFields(data interface{}, handler ...fieldHandler) error {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)

	for t.Kind() == reflect.Pointer { // if
		t = t.Elem()
		v = v.Elem()
	}

	//if t.Kind() != reflect.Struct {
	//	panic("data is not a struct")
	//}

	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)

		for _, h := range handler {
			tagValue := ft.Tag.Get(h.tag)

			if tagValue == "-" || tagValue == "" {
				continue
			}

			if !ft.IsExported() {
				panic(fmt.Sprintf("field [%s] is unexported", ft.Name))
			}

			var target reflect.Value

			switch ft.Type.Kind() {
			case reflect.Pointer:
				// if fv.IsNil() {
				// 	refValue := reflect.New(ft.Type.Elem())
				// 	fv.Set(refValue)
				// 	continue // TODO ...
				// }

				target = fv
			case reflect.Struct:
				err := handleStructFields(fv.Addr().Interface(), handler...)
				if err != nil {
					return err
				}

				continue
			default:
				target = fv.Addr()
			}

			err := h.fn(tagValue, target.Interface())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func parseRequest(_ context.Context, data interface{}, r *http.Request, argsPlace []string, args []string) interface{} {
	contentType := r.Header.Get("Content-Type")

	switch {
	case contentType == "application/json":
		err := json.NewDecoder(r.Body).Decode(data)

		if err != nil && err != io.EOF {
			//return NewResponse(response{
			//	Status:      http.StatusBadRequest,
			//	Code:        "incorrect-json-data",
			//	Description: err.Error(),
			//}).SetCode(http.StatusBadRequest)
		}
	case strings.HasPrefix(contentType, "multipart/form-data"):
		//mp, err := r.MultipartReader()
		//
		//if err != nil {
		//	//return httpserver.NewResponse(response{
		//	//	Status:      http.StatusBadRequest,
		//	//	Code:        "incorrect-form-data-data",
		//	//	Description: err.Error(),
		//	//}).SetCode(http.StatusBadRequest)
		//}
		//
		//if data, ok := data.(interface {
		//	SetMultipart(*multipart.Reader)
		//}); ok {
		//	data.SetMultipart(mp)
		//}
	}

	err := handleStructFields(data,
		fieldHandler{"query", func(tag string, v interface{}) error {
			return parseQuery(tag, v, r.URL)
		}},
		fieldHandler{"args", func(tag string, v interface{}) error {
			return parseArgs(tag, v, argsPlace, args)
		}},
		fieldHandler{"json", trim},
	)

	if err != nil {
		//return handlerResult(ctx, err)
	}

	//err = data.Validate(r, args)
	//if err != nil {
	//	return handlerResult(ctx, err)
	//}

	return nil
}

func Create[RQ, RP, A, C any](fn func(context.Context, C, A, RQ) (RP, error), options ...Option) *MethodHandler[C, A] {
	var (
		rq RQ
		rs RP
	)

	rqRef := reflect.TypeOf(rq).Elem()
	rpRef := reflect.TypeOf(rs).Elem()

	params := parameters{}

	rqType := definitionFromObject(rqRef, &params, "")
	rpType := definitionFromObject(rpRef, &params, "")

	handler := &MethodHandler[C, A]{
		description: apiDescription{
			headers: params.headers,
			args:    params.args,
			query:   params.query,

			requestObject: &objectType{
				name:   rqRef.Name(),
				object: &rqType,
			},

			responseObject: &objectType{
				name:        rpRef.Name(),
				description: "Success response",
				object:      &rpType,
			},
		},
		handlerFunc: func(ctx context.Context, c C, a A, r *http.Request, argsPlace []string, args []string) interface{} {
			var request RQ

			t := reflect.TypeOf(request).Elem()
			request = reflect.New(t).Interface().(RQ)

			res := parseRequest(ctx, request, r, argsPlace, args)
			if res != nil {
				return res
			}

			result, err := fn(ctx, c, a, request)
			if err != nil {
				return err
			}

			return result
		},
	}

	for _, o := range options {
		o(&handler.description)
	}

	return handler
}

type OrderedMapField[T any] struct {
	name  string
	value T
}

type OrderedMap[T any] []OrderedMapField[T]

func (p *OrderedMap[T]) Add(name string, t T) {
	for i, v := range *p {
		if v.name == name {
			(*p)[i].value = t
			return
		}
	}

	*p = append(*p, OrderedMapField[T]{
		name:  name,
		value: t,
	})
}

func (p *OrderedMap[T]) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	buf.Write([]byte{'{'})

	for i, v := range *p {
		b, err := json.Marshal(&v.value)
		if err != nil {
			return nil, err
		}

		_, _ = fmt.Fprintf(buf, "%q:", v.name)
		buf.Write(b)
		if i < len(*p)-1 {
			buf.Write([]byte{','})
		}
	}

	buf.Write([]byte{'}'})

	return buf.Bytes(), nil
}

func intFormat(k reflect.Kind) string {
	switch k {
	case reflect.Int: // TODO ...
		return "int64"
	case reflect.Int8:
		return "int8"
	case reflect.Int16:
		return "int16"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint: // TODO ...
		return "uint64"
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	default:
		return ""
	}
}

type parameters struct {
	headers, args, query OrderedMap[apiType]
}

func definitionFromObject(t reflect.Type, p *parameters, desc string) apiType {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		fields := make(OrderedMap[apiType], 0, t.NumField())

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			dd := f.Tag.Get("desc")

			if h := f.Tag.Get("header"); h != "" {
				p.headers.Add(h, apiType{
					Type:        TypeString, // TODO
					Description: dd,
					Required:    false,
				})

				continue
			}

			if h := f.Tag.Get("args"); h != "" {
				p.args.Add(h, apiType{
					Type:        TypeString, // TODO
					Description: dd,
					Required:    true,
				})

				continue
			}

			if h := f.Tag.Get("query"); h != "" {
				p.query.Add(h, apiType{
					Type:        TypeString, // TODO
					Description: dd,
					Required:    false,
				})

				continue
			}

			jsonTag := f.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = f.Name
			}

			fields.Add(jsonTag, definitionFromObject(f.Type, p, dd))
		}

		return apiType{
			Type:        TypeObject,
			Description: desc,
			Properties:  fields,
		}
	case reflect.String:
		return apiType{
			Type:        TypeString,
			Description: desc,
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return apiType{
			Type:        TypeInteger,
			Description: desc,
			Format:      intFormat(t.Kind()),
		}
	case reflect.Bool:
		return apiType{
			Type:        TypeBool,
			Description: desc,
		}
	default:
		return apiType{}
	}
}
