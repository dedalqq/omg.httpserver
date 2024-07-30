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

func AuthOptional() Option {
	return func(d *apiDescription) {
		d.authOptional = true
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

func setValue(targetValue interface{}, value string) error {
	//if reflect.TypeOf(targetValue).Kind() != reflect.Pointer {
	//	return fmt.Errorf("value ", value)
	//}

	switch v := targetValue.(type) {
	case *string:
		*v = value
		return nil

	case *int, *int8, *int16, *int32, *int64:
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("value [%s] must be int", value)
		}

		reflect.ValueOf(targetValue).Elem().SetInt(int64(intVal))

		return nil

	case *uint, *uint8, *uint16, *uint32, *uint64:
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("value [%s] must be uint", value)
		}

		reflect.ValueOf(targetValue).Elem().SetUint(uint64(intVal))

		return nil
	case *bool:
		switch value {
		case "true", "True", "1":
			*v = true
		case "false", "False", "0":
			*v = false
		default:
			return fmt.Errorf("value [%s] must be bool", value)
		}

		return nil
	default:
		return json.Unmarshal([]byte(value), targetValue)
	}
}

func parseArgs(tagValue string, targetValue interface{}, argsPlace []string, args []string) error {
	var argValue string

	for i, name := range argsPlace {
		if name == tagValue {
			argValue = args[i]
		}
	}

	return setValue(targetValue, argValue)
}

func parseQuery(tagValue string, targetValue interface{}, url *url.URL) error {
	queryValue := url.Query().Get(tagValue)
	if queryValue == "" {
		return nil
	}

	return setValue(targetValue, queryValue)
}

func parseHeader(tagValue string, targetValue interface{}, header http.Header) error {
	headerValue := header.Get(tagValue)
	if headerValue == "" {
		return nil
	}

	return setValue(targetValue, headerValue)
}

func handleStructFields(data interface{}, request *http.Request, handler ...fieldHandler) error {
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

		if _, ok := fv.Interface().(*url.URL); ok { // TODO refactor
			fv.Set(reflect.ValueOf(request.URL))
		}

		for _, h := range handler {
			tagValue := ft.Tag.Get(h.tag)

			if tagValue == "-" || tagValue == "" || !ft.IsExported() {
				continue
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
				err := handleStructFields(fv.Addr().Interface(), request, handler...)
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

	err := handleStructFields(data, r,
		fieldHandler{"header", func(tag string, v interface{}) error { return parseHeader(tag, v, r.Header) }},
		fieldHandler{"query", func(tag string, v interface{}) error { return parseQuery(tag, v, r.URL) }},
		fieldHandler{"args", func(tag string, v interface{}) error { return parseArgs(tag, v, argsPlace, args) }},

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
	var rq RQ
	var rp RP

	rqRef := reflect.TypeOf(rq)
	rpRef := reflect.TypeOf(rp)

	var rqName, rpName string

	if rqRef != nil {
		for rqRef.Kind() == reflect.Pointer {
			rqRef = rqRef.Elem()
		}

		rqName = rqRef.Name()
	}

	if rpRef != nil {
		for rpRef.Kind() == reflect.Pointer {
			rpRef = rpRef.Elem()
		}

		rpName = rpRef.Name()
	}

	successStatusCode := http.StatusOK

	if t, ok := (interface{})(rp).(ResponseWithCode); ok {
		successStatusCode = t.Code()
	}

	params := parameters{}

	var rqType, rpType *apiType

	//rqType = definitionFromObject(rqRef, &params, "") TODO support recursive schema

	switch (interface{})(rp).(type) {
	case NoContent, *Swagger:
	default:
		//rpType = definitionFromObject(rpRef, &params, "") TODO support recursive schema
	}

	handler := &MethodHandler[C, A]{
		description: apiDescription{
			headers: params.headers,
			args:    params.args,
			query:   params.query,

			requestObject: objectType{
				name:   rqName,
				object: rqType,
			},

			successStatusCode: successStatusCode,

			responseObject: objectType{
				name:        rpName,
				description: "Success response",
				object:      rpType,
			},
		},

		handlerFunc: func(ctx context.Context, c C, a A, r *http.Request, argsPlace []string, args []string) interface{} {
			var request RQ

			t := reflect.TypeOf(request)

			if t != nil {
				if t.Kind() == reflect.Pointer { // TODO try support not pointer
					request = reflect.New(t.Elem()).Interface().(RQ)
				}

				res := parseRequest(ctx, request, r, argsPlace, args)
				if res != nil {
					return res
				}
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

func definitionFromObject(t reflect.Type, p *parameters, desc string) *apiType {
	if t == nil {
		return nil
	}

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

			parameter := definitionFromObject(f.Type, p, dd)
			if parameter != nil {
				fields.Add(jsonTag, *parameter)
			}
		}

		return &apiType{
			Type:        TypeObject,
			Description: desc,
			Properties:  fields,
		}
	case reflect.Slice:
		return &apiType{
			Type:        TypeArray,
			Description: desc,
			Items:       definitionFromObject(t.Elem(), p, ""),
		}
	case reflect.String:
		return &apiType{
			Type:        TypeString,
			Description: desc,
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &apiType{
			Type:        TypeInteger,
			Description: desc,
			Format:      intFormat(t.Kind()),
		}
	case reflect.Bool:
		return &apiType{
			Type:        TypeBool,
			Description: desc,
		}
	case reflect.Map:
		return &apiType{
			Type:        TypeObject,
			Description: desc,
		}
	default:
		return nil
	}
}
