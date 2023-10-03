package httpserver

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type route[C, A any] struct {
	pattern *regexp.Regexp
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
	path = strings.ReplaceAll(path, "{any}", "([^/]+)")
	path = strings.ReplaceAll(path, "{int}", "(\\d+)")
	path = strings.ReplaceAll(path, "{uuid}", "([0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12})")

	reg, err := regexp.Compile(fmt.Sprintf("^%s$", path))
	if err != nil {
		panic(err)
	}

	r.routes = append(r.routes, route[C, A]{
		pattern: reg,
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

func (r *Router[C, A]) get(path string) (*Handler[C, A], []string) {
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
