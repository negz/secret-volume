package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type HTTPRouter interface {
	Handler(method, path string, handler http.Handler)
	GET(path string, handler http.Handler)
	POST(path string, handler http.Handler)
	DELETE(path string, handler http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetParam(r *http.Request, key string) string
}

type hrParams struct{}

// A httprouter based HTTP router!
type hrHTTPRouter struct {
	hr *httprouter.Router
	k  hrParams
}

type HRHTTPRouterOption func(*hrHTTPRouter) error

func Router(r *httprouter.Router) HRHTTPRouterOption {
	return func(h *hrHTTPRouter) error {
		h.hr = r
		return nil
	}
}

func NewHRHTTPRouter(ro ...HRHTTPRouterOption) (HTTPRouter, error) {
	r := &hrHTTPRouter{httprouter.New(), hrParams{}}
	for _, o := range ro {
		if err := o(r); err != nil {
			return nil, err
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
