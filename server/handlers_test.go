package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
)

type noopVolumeManager struct{}

func (v *noopVolumeManager) Create(_ *api.Volume) error {
	return nil
}

func (v *noopVolumeManager) Destroy(id string) error {
	return nil
}

func (v *noopVolumeManager) Get(id string) (*api.Volume, error) {
	return fixtures.TestVolume, nil
}

func (v *noopVolumeManager) List() (api.Volumes, error) {
	return fixtures.TestVolumes, nil
}

func (v *noopVolumeManager) MetadataFile() string {
	return ".meta"
}

func TestHTTPHandlers(t *testing.T) {
	h, err := NewHTTPHandlers(&noopVolumeManager{})
	if err != nil {
		t.Errorf("NewHTTPHandlers(): %v", err)
		return
	}

	h.setupRoutes()

	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

		if w.Code != http.StatusOK {
			t.Errorf("w.Code want %v, got %v (%v)", http.StatusOK, w.Code, w.Body.String())
			return
		}

		vs, err := api.ReadVolumesJSON(w.Body)
		if err != nil {
			t.Errorf("api.ReadVolumesJSON(%v): %v", w.Body, err)
			return
		}

		if !reflect.DeepEqual(vs, fixtures.TestVolumes) {
			t.Errorf("Wanted %v, got %v", fixtures.TestVolumes, vs)
		}
	})
	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.r.ServeHTTP(w, httptest.NewRequest("GET", "/id", nil))

		if w.Code != http.StatusOK {
			t.Errorf("w.Code want %v, got %v (%v)", http.StatusOK, w.Code, w.Body.String())
			return
		}

		v, err := api.ReadVolumeJSON(w.Body)
		if err != nil {
			t.Errorf("api.ReadVolumeJSON(%v): %v", w.Body, err)
			return
		}

		if !reflect.DeepEqual(v, fixtures.TestVolume) {
			t.Errorf("Wanted %v, got %v", fixtures.TestVolume, v)
		}
	})
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.r.ServeHTTP(w, httptest.NewRequest("DELETE", "/id", nil))

		if w.Code != http.StatusOK {
			t.Errorf("w.Code want %v, got %v (%v)", http.StatusOK, w.Code, w.Body.String())
			return
		}
	})
	t.Run("Create", func(t *testing.T) {
		b := &bytes.Buffer{}
		fixtures.TestVolume.WriteJSON(b)

		w := httptest.NewRecorder()
		h.r.ServeHTTP(w, httptest.NewRequest("POST", "/", b))

		if w.Code != http.StatusOK {
			t.Errorf("w.Code want %v, got %v (%v)", http.StatusOK, w.Code, w.Body.String())
			return
		}

		v, err := api.ReadVolumeJSON(w.Body)
		if err != nil {
			t.Errorf("api.ReadVolumeJSON(%v): %v", w.Body, err)
			return
		}

		if !reflect.DeepEqual(v, fixtures.TestVolume) {
			t.Errorf("Wanted %v, got %v", fixtures.TestVolume, v)
		}
	})
}
