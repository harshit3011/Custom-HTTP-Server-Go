
package router

import (
	"fmt"
	"strings"

	"github.com/harshit3011/Custom-HTTP-Server-Go/parser"
	"github.com/harshit3011/Custom-HTTP-Server-Go/response"
)

type Handler func(*response.ResponseWriter, *parser.Request)
type Middleware func(Handler) Handler

type Router struct {
	routes     map[string]map[string]Handler
	middleware []Middleware
}

func New() *Router {
	return &Router{
		routes: make(map[string]map[string]Handler),
	}
}

func (r *Router) Handle(method, path string, handler Handler) error {
	method = strings.ToUpper(method)

	if !isToken(method) || path == "" || !strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid route: %s %s", method, path)
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	if r.routes[path] == nil {
		r.routes[path] = make(map[string]Handler)
	}

	if _, exists := r.routes[path][method]; exists {
		return fmt.Errorf("route already registered: %s %s", method, path)
	}

	r.routes[path][method] = handler
	return nil
}

func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) Serve(req *parser.Request) *response.ResponseWriter {
	w := response.NewResponseWriter()

	path := strings.SplitN(req.Target, "?", 2)[0]
	method := strings.ToUpper(req.Method)

	handlers, pathExists := r.routes[path]
	handler, routeExists := handlers[method]

	if !routeExists {
		if pathExists {
			_ = w.Header().Set("Allow", allowedMethods(handlers))
			w.Text(405, "Method Not Allowed\n")
		} else {
			w.Text(404, "Not Found\n")
		}

		return w
	}

	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	handler(w, req)
	return w
}

func allowedMethods(handlers map[string]Handler) string {
	methods := make([]string, 0, len(handlers))

	for method := range handlers {
		methods = append(methods, method)
	}

	return strings.Join(methods, ", ")
}

func isToken(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') {
			continue
		}

		if strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			continue
		}

		return false
	}

	return true
}
