package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// A HTTPRouter provides a HTTP request router (aka) mux implementation.
type HTTPRouter interface {
	// Handler attaches the supplied Handler to the supplied HTTP method and
	// path.
	Handler(method, path string, handler http.Handler)
	// GET is a convenience wrapper around Handler.
	GET(path string, handler http.Handler)
	// POST is a convenience wrapper around Handler.
	POST(path string, handler http.Handler)
	// DELETE is a convenience wrapper around Handler.
	DELETE(path string, handler http.Handler)
	// ServeHTTP allows the use of a HTTPRouter as a http.Handler.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	// GetParam returns a parameter encoded within a path handled by Handler.
	// The syntax for encoding a parameter in the path will vary depending on
	// the underlying routing implementation, but it might (read: does) work
	// as per the https://github.com/julienschmidt/httprouter implementation.
	// i.e. If your route path is defined as /things/:thing GetParam(thing) will
	// return myawesomething given a Request for /things/myawesomething.
	GetParam(r *http.Request, key string) string
}

type hrParams struct{}

// A httprouter based HTTP router!
type hrHTTPRouter struct {
	hr *httprouter.Router
	k  hrParams
}

// A HRHTTPRouterOption represents an argument to NewHRHTTPRouter.
type HRHTTPRouterOption func(*hrHTTPRouter) error

// Router provides an alternative Router to be used by NewHRHTTPRouter. A router
// returned by httprouter.New() is used by default.
func Router(r *httprouter.Router) HRHTTPRouterOption {
	return func(h *hrHTTPRouter) error {
		h.hr = r
		return nil
	}
}

// NewHRHTTPRouter creates a new HTTPRouter backed by
// https://github.com/julienschmidt/httprouter.
func NewHRHTTPRouter(ro ...HRHTTPRouterOption) (HTTPRouter, error) {
	r := &hrHTTPRouter{httprouter.New(), hrParams{}}
	for _, o := range ro {
		if err := o(r); err != nil {
			return nil, errors.Wrap(err, "cannot apply HTTP router option")
		}
	}
	return r, nil
}

func (r *hrHTTPRouter) params(rq *http.Request) httprouter.Params {
	if p, ok := rq.Context().Value(r.k).(httprouter.Params); ok {
		return p
	}
	return nil
}

func (r *hrHTTPRouter) GetParam(rq *http.Request, name string) string {
	if p := r.params(rq); p != nil {
		return p.ByName(name)
	}
	return ""
}

func (r *hrHTTPRouter) Handler(m, p string, h http.Handler) {
	r.hr.Handle(m, p, func(w http.ResponseWriter, rq *http.Request, p httprouter.Params) {
		c := rq.Context()
		c = context.WithValue(c, r.k, p)
		rq = rq.WithContext(c)
		h.ServeHTTP(w, rq)
	})
}

func (r *hrHTTPRouter) GET(p string, h http.Handler) {
	r.Handler("GET", p, h)
}

func (r *hrHTTPRouter) POST(p string, h http.Handler) {
	r.Handler("POST", p, h)
}

func (r *hrHTTPRouter) DELETE(p string, h http.Handler) {
	r.Handler("DELETE", p, h)
}

func (r *hrHTTPRouter) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	r.hr.ServeHTTP(w, rq)
}
