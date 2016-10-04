package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var hrHttpRouterTests = []struct {
	route string
	url   string
	k     string
	v     string
}{
	{"/derps/:herp", "/derps/derp", "herp", "derp"},
	{"/:womp", "/wamp", "womp", "wamp"},
	{"/thing/:exists", "/thing/thang", "doesnotexist", ""},
}

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}
func TestHRHttpRouter(t *testing.T) {
	for _, tt := range hrHttpRouterTests {
		r, err := NewHRHttpRouter()
		if err != nil {
			t.Errorf("NewHRHttpRouter(): %v", err)
			continue
		}
		t.Run("GetParam", func(t *testing.T) {
			r.GET(tt.route, http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
				if a := r.GetParam(rq, tt.k); a != tt.v {
					t.Errorf("r.GetParam(%v): Want %v, got %v", tt.k, tt.v, a)
				}
			}))

			r.ServeHTTP(&mockResponseWriter{}, httptest.NewRequest("GET", tt.url, nil))
		})
	}
}
