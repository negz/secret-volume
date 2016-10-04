package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type HttpRouter interface {
	Handler(method, path string, handler http.Handler)
	GET(path string, handler http.Handler)
	POST(path string, handler http.Handler)
	DELETE(path string, handler http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetParam(r *http.Request, key string) string
}

type hrParams struct{}

// A httprouter based HTTP router!
type hrHttpRouter struct {
	hr *httprouter.Router
	k  hrParams
}

type HRHttpRouterOption func(*hrHttpRouter) error

func Router(r *httprouter.Router) HRHttpRouterOption {
	return func(h *hrHttpRouter) error {
		h.hr = r
		return nil
	}
}

func NewHRHttpRouter(ro ...HRHttpRouterOption) (HttpRouter, error) {
	r := &hrHttpRouter{httprouter.New(), hrParams{}}
	for _, o := range ro {
		if err := o(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *hrHttpRouter) params(rq *http.Request) httprouter.Params {
	if p, ok := rq.Context().Value(r.k).(httprouter.Params); ok {
		return p
	} else {
		return nil
	}
}

func (r *hrHttpRouter) GetParam(rq *http.Request, name string) string {
	if p := r.params(rq); p != nil {
		return p.ByName(name)
	}
	return ""
}

func (r *hrHttpRouter) Handler(m, p string, h http.Handler) {
	r.hr.Handle(m, p, func(w http.ResponseWriter, rq *http.Request, p httprouter.Params) {
		c := rq.Context()
		c = context.WithValue(c, r.k, p)
		rq = rq.WithContext(c)
		h.ServeHTTP(w, rq)
	})
}

func (r *hrHttpRouter) GET(p string, h http.Handler) {
	r.Handler("GET", p, h)
}

func (r *hrHttpRouter) POST(p string, h http.Handler) {
	r.Handler("POST", p, h)
}

func (r *hrHttpRouter) DELETE(p string, h http.Handler) {
	r.Handler("DELETE", p, h)
}

func (r *hrHttpRouter) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	r.hr.ServeHTTP(w, rq)
}
