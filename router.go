package httpserver

import (
	"context"
	"fmt"
	"path"
	"regexp"
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

func descriptionHandler[C, A any](handler *MethodHandler[C, A], definitions *OrderedMap[apiType]) *apiHandler {
	if handler == nil {
		return nil
	}

	descHandler := &apiHandler{}

	appendParameters(&descHandler.Parameters, handler.description.headers, "header")
	appendParameters(&descHandler.Parameters, handler.description.args, "path")
	appendParameters(&descHandler.Parameters, handler.description.query, "query")

	if obj := handler.description.requestObject; obj != nil {
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

	if obj := handler.description.responseObject; obj != nil {
		definitions.Add(obj.name, *obj.object)

		descHandler.Responses.Add("200", apiParameter{
			Description: obj.description,
			Schema: &apiSchema{
				Ref: fmt.Sprintf("#/definitions/%s", obj.name),
			},
		})
	}

	return descHandler
}

func (r *Router[C, A]) renderSwagger(_ context.Context, _ C, _ A, _ struct{}) (*Swagger, error) {
	swagger := &Swagger{
		Swagger: "2.0",
		Info: apiInfo{
			Title:       "Some API",
			Version:     "v0.0",
			Description: "Auto generated documentation",
		},
	}

	for _, rt := range r.routes {
		swagger.Paths.Add(rt.path, apiEndpoint{
			Get:    descriptionHandler(rt.handler.Get, &swagger.Definitions),
			Post:   descriptionHandler(rt.handler.Post, &swagger.Definitions),
			Put:    descriptionHandler(rt.handler.Put, &swagger.Definitions),
			Delete: descriptionHandler(rt.handler.Delete, &swagger.Definitions),
			Patch:  descriptionHandler(rt.handler.Patch, &swagger.Definitions),
		})
	}

	return swagger, nil
}

func (r *Router[C, A]) AddSwagger(path string) {
	r.Add(path, Handler[C, A]{
		Get: Create(r.renderSwagger),
	})
}

//func (r *SubRouter[C, A]) AddSubSwagger(path string, sr *SubRouter[C, A]) {
//
//}

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
