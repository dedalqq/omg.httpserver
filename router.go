package httpserver

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var placeholderReg *regexp.Regexp

func init() {
	var err error

	placeholderReg, err = regexp.Compile("{([^/]+)}")
	if err != nil {
		panic(err)
	}
}

type route[C, A any] struct {
	path    string
	pattern *regexp.Regexp
	args    []string
	handler Handler[C, A]
}

// Router is a router object
type Router[C, A any] struct {
	routes       []route[C, A]
	defaultRoute *Handler[C, A]
}

// SubRouter is a sub router object
type SubRouter[C, A any] struct {
	r       *Router[C, A]
	subPath string
}

// NewRouter creates and returns new router
func NewRouter[C, A any]() Router[C, A] {
	return Router[C, A]{}
}

// Add new router rule in to router object for handler
func (r *Router[C, A]) Add(path string, h Handler[C, A]) *Router[C, A] {
	res := placeholderReg.FindAllStringSubmatch(path, -1)
	args := make([]string, 0, len(res))
	for _, r := range res {
		args = append(args, r[1])
	}

	reg, err := regexp.Compile(fmt.Sprintf("^%s$", placeholderReg.ReplaceAllString(path, "([^/]+)")))
	if err != nil {
		panic(err) // TODO
	}

	r.routes = append(r.routes, route[C, A]{
		path:    path,
		pattern: reg,
		args:    args,
		handler: h,
	})

	return r
}

// Default sets a default handler for handle request if route rule was not found
func (r *Router[C, A]) Default(h Handler[C, A]) *Router[C, A] {
	r.defaultRoute = &h

	return r
}

// SubRoute returns new sub route object for add rules in sub root
func (r *Router[C, A]) SubRoute(subPath string) *SubRouter[C, A] {
	return &SubRouter[C, A]{
		r:       r,
		subPath: subPath,
	}
}

// Add new router rule in to router object for handler
func (r *SubRouter[C, A]) Add(subPath string, h Handler[C, A]) *SubRouter[C, A] {
	r.r.Add(path.Join(r.subPath, subPath), h)

	return r
}

func appendParameters(params *[]apiParameter, values OrderedMap[apiType], in string) {
	for _, v := range values {
		*params = append(*params, apiParameter{
			In:       in,
			Name:     v.name,
			Type:     v.value.Type,
			Required: v.value.Required,
		})
	}
}

func descriptionHandler[C, A any](handler *MethodHandler[C, A], definitions *OrderedMap[apiType], withBody bool) *apiHandler {
	if handler == nil {
		return nil
	}

	descHandler := &apiHandler{}

	appendParameters(&descHandler.Parameters, handler.description.headers, "header")
	appendParameters(&descHandler.Parameters, handler.description.args, "path")
	appendParameters(&descHandler.Parameters, handler.description.query, "query")

	obj := handler.description.requestObject

	if withBody && len(obj.object.Properties) > 0 {
		definitions.Add(obj.name, *obj.object)

		descHandler.Parameters = append(descHandler.Parameters, apiParameter{
			In:       "body",
			Name:     obj.name,
			Required: true,
			Schema: &apiSchema{
				Ref: fmt.Sprintf("#/definitions/%s", obj.name),
			},
		})
	}

	respDefinition := apiParameter{
		Description: handler.description.responseObject.description,
	}

	if obj := handler.description.responseObject; obj.object != nil {
		definitions.Add(obj.name, *obj.object)

		respDefinition.Schema = &apiSchema{
			Ref: fmt.Sprintf("#/definitions/%s", obj.name),
		}
	}

	descHandler.Responses.Add(strconv.Itoa(handler.description.successStatusCode), respDefinition)

	return descHandler
}

func (r *Router[C, A]) renderSwagger(prefix string, opt SwaggerOpt) func(_ context.Context, _ C, _ A, _ struct{}) (*Swagger, error) {
	opt.fillDefault(prefix)

	return func(_ context.Context, _ C, _ A, _ struct{}) (*Swagger, error) {
		swagger := &Swagger{
			Swagger: "2.0",
			Info: apiInfo{
				Title:       opt.Title,
				Version:     opt.Version,
				Description: opt.Description,
			},
			BasePath: opt.BasePath,
		}

		for _, rt := range r.routes {
			if !strings.HasPrefix(rt.path, prefix) {
				continue
			}

			pp := rt.path[len(prefix):]

			if pp == "" {
				pp = "/"
			}

			swagger.Paths.Add(pp, apiEndpoint{
				Get:    descriptionHandler(rt.handler.Get, &swagger.Definitions, false),
				Post:   descriptionHandler(rt.handler.Post, &swagger.Definitions, true),
				Put:    descriptionHandler(rt.handler.Put, &swagger.Definitions, true),
				Delete: descriptionHandler(rt.handler.Delete, &swagger.Definitions, true),
				Patch:  descriptionHandler(rt.handler.Patch, &swagger.Definitions, true),
			})
		}

		return swagger, nil
	}
}

type SwaggerOpt struct {
	Title       string
	Version     string
	Description string

	BasePath string
}

func (o *SwaggerOpt) fillDefault(prefix string) {
	if o.Title == "" {
		o.Title = "Some API"
	}

	if o.Version == "" {
		o.Version = "v0.0"
	}

	if o.Description == "" {
		o.Description = "Auto generated documentation"
	}

	if o.BasePath == "" {
		o.BasePath = prefix
	}
}

func (r *Router[C, A]) AddSwagger(path string, opt SwaggerOpt) {
	r.Add(path, Handler[C, A]{
		Get: Create(r.renderSwagger("", opt)),
	})
}

func (r *Router[C, A]) AddSwaggerSubRoute(path string, sr *SubRouter[C, A], opt SwaggerOpt) {
	r.Add(path, Handler[C, A]{
		Get: Create(r.renderSwagger(sr.subPath, opt)),
	})
}

func (r *Router[C, A]) get(path string) (*Handler[C, A], []string, []string) {
	for _, r := range r.routes {
		if res := r.pattern.FindStringSubmatch(path); len(res) > 0 {
			return &r.handler, r.args, res[1:]
		}
	}

	if r.defaultRoute != nil {
		return r.defaultRoute, nil, nil
	}

	return nil, nil, nil
}
