package httpserver

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type route struct {
	pattern *regexp.Regexp
	handler Handler
}

// Router is a router object
type Router struct {
	routes       []route
	defaultRoute *Handler
}

// SubRouter is a sub router object
type SubRouter struct {
	r       *Router
	subPath string
}

// NewRouter creates and returns new router
func NewRouter() Router {
	return Router{}
}

// Add add new router rule in to router object for handler
func (r *Router) Add(path string, h Handler) *Router {
	path = strings.ReplaceAll(path, "{any}", "([^/]+)")
	path = strings.ReplaceAll(path, "{int}", "(\\d+)")
	path = strings.ReplaceAll(path, "{uuid}", "([0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12})")

	reg, err := regexp.Compile(fmt.Sprintf("^%s$", path))
	if err != nil {
		panic(err)
	}

	r.routes = append(r.routes, route{
		pattern: reg,
		handler: h,
	})

	return r
}

// Default sets a default handler for handle request if route rule was not found
func (r *Router) Default(h Handler) *Router {
	r.defaultRoute = &h

	return r
}

// SubRoute returns new sub route object for add rules in sub root
func (r *Router) SubRoute(subPath string) *SubRouter {
	return &SubRouter{
		r:       r,
		subPath: subPath,
	}
}

// Add add new router rule in to router object for handler
func (r *SubRouter) Add(subPath string, h Handler) *SubRouter {
	r.r.Add(path.Join(r.subPath, subPath), h)

	return r
}

func (r *Router) get(path string) (*Handler, []string) {
	for _, r := range r.routes {
		if res := r.pattern.FindStringSubmatch(path); len(res) > 0 {
			return &r.handler, res[1:]
		}
	}

	if r.defaultRoute != nil {
		return r.defaultRoute, []string{}
	}

	return nil, nil
}
