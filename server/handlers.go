package server

import (
	"fmt"
	"net/http"

	"github.com/uber-go/zap"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/volume"
)

type HttpHandlers struct {
	v     volume.VolumeManager
	r     HttpRouter
	idKey string
}

type HttpHandlersOption func(*HttpHandlers) error

func HttpHandlersRouter(r HttpRouter) HttpHandlersOption {
	return func(h *HttpHandlers) error {
		h.r = r
		return nil
	}
}

func NewHttpHandlers(v volume.VolumeManager, ho ...HttpHandlersOption) (*HttpHandlers, error) {
	r, err := NewHRHttpRouter()
	if err != nil {
		return nil, err
	}
	s := &HttpHandlers{v, r, "id"}
	for _, o := range ho {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (h *HttpHandlers) setupRoutes() {
	h.r.GET("/", http.HandlerFunc(logReq(json(h.list))))
	h.r.POST("/", http.HandlerFunc(logReq(json(h.create))))
	h.r.GET("/:id", http.HandlerFunc(logReq(json(h.ensureParam(h.get, h.idKey)))))
	h.r.DELETE("/:id", http.HandlerFunc(logReq(h.ensureParam(h.delete, h.idKey))))
}

func (h *HttpHandlers) HttpServer(addr string) *http.Server {
	h.setupRoutes()
	return &http.Server{Addr: addr, Handler: h.r}
}

func (h *HttpHandlers) list(w http.ResponseWriter, _ *http.Request) {
	vs, err := h.v.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := vs.WriteJSON(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *HttpHandlers) get(w http.ResponseWriter, r *http.Request) {
	id := h.r.GetParam(r, h.idKey)
	v, err := h.v.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := v.WriteJSON(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *HttpHandlers) create(w http.ResponseWriter, r *http.Request) {
	v, err := api.ReadVolumeJSON(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.v.Create(v); err != nil {
		// TODO(negz): This is just as likely to be StatusBadRequest (i.e. bad certificate)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reserialise (rather than return the sent copy) to strip out the keypair,
	// which does not get returned in subsequent queries.
	if err := v.WriteJSON(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *HttpHandlers) delete(w http.ResponseWriter, r *http.Request) {
	id := h.r.GetParam(r, h.idKey)
	if err := h.v.Destroy(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func (h *HttpHandlers) ensureParam(fn http.HandlerFunc, p string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.r.GetParam(r, p) == "" {
			http.Error(w, fmt.Sprintf("Missing URL component: %v", p), http.StatusBadRequest)
			return
		}
		fn(w, r)
	}
}

func logReq(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO(negz): Wrap w so we can log our response.
		log.Info("http request",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("addr", r.RemoteAddr))
		fn(w, r)
	}
}

func json(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}
}
