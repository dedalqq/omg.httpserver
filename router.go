package httpServer

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type Router struct {
	routes       map[*regexp.Regexp]Handler
	defaultRoute *Handler
}

type SubRouter struct {
	r       *Router
	subPath string
}

func NewRouter() Router {
	return Router{
		routes: make(map[*regexp.Regexp]Handler),
	}
}

func (r *Router) Add(path string, h Handler) *Router {
	path = strings.ReplaceAll(path, "{any}", "(.+)")
	path = strings.ReplaceAll(path, "{uuid}", "([0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12})")

	reg, err := regexp.Compile(fmt.Sprintf("^%s$", path))
	if err != nil {
		panic(err)
	}

	r.routes[reg] = h

	return r
}

func (r *Router) Default(h Handler) *Router {
	r.defaultRoute = &h

	return r
}

func (r *Router) SubRoute(subPath string) SubRouter {
	return SubRouter{
		r:       r,
		subPath: subPath,
	}
}

func (r *SubRouter) Add(subPath string, h Handler) *SubRouter {
	r.r.Add(path.Join(r.subPath, subPath), h)

	return r
}

func (r *Router) get(path string) (*Handler, []string) {
	for r, h := range r.routes {
		if res := r.FindStringSubmatch(path); len(res) > 0 {
			return &h, res[1:]
		}
	}

	if r.defaultRoute != nil {
		return r.defaultRoute, []string{}
	}

	return nil, nil
}
