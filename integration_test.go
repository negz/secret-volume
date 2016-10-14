// +build integration

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/spf13/afero"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
	"github.com/negz/secret-volume/secrets"
	"github.com/negz/secret-volume/server"
	"github.com/negz/secret-volume/volume"
)

func localhostWithRandomPort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

func TestIntegration(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := volume.NewNoopMounter("/secrets")

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		z, err := afero.NewOsFs().Open("fixtures/yaml.tar.gz")
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		io.Copy(w, z)
		z.Close()
	}))
	ts.StartTLS()
	defer ts.Close()

	lb, _ := fixtures.PredictableLoadBalancerFor(ts.URL)

	sp, _ := secrets.NewTalosProducer(lb)
	sps := map[api.SecretSource]secrets.Producer{api.TalosSecretSource: sp}

	vm, _ := volume.NewManager(m, sps, volume.Filesystem(fs), volume.WriteJSONSecrets("secrets.json"))

	h, _ := server.NewHTTPHandlers(vm)

	addr, err := localhostWithRandomPort()
	if err != nil {
		t.Fatalf("localhostWithRandomPort(): %v", err)
	}

	go h.HTTPServer(addr).ListenAndServe()

	v, _ := fixtures.TestVolumeWithCert("fixtures/cert.pem", "fixtures/key.pem")

	t.Run("Create", func(t *testing.T) {
		url := fmt.Sprintf("http://%v", addr)
		cnt := "application/json"

		// We need an 'insecure' volume that will JSON encode its KeyPair
		iv := fixtures.NewInsecureVolume(v)
		b := &bytes.Buffer{}
		err := iv.WriteJSON(b)

		r, err := http.Post(url, cnt, b)
		if err != nil {
			t.Fatalf("http.Post(%v, %v, %v): %v", url, cnt, b, err)
		}

		if r.StatusCode != http.StatusOK {
			e, _ := ioutil.ReadAll(r.Body)
			t.Fatalf("http.Post(%v, %v, %v): Want %v, got %v: %s", url, cnt, b, http.StatusOK, r.StatusCode, e)
		}

		got, err := api.ReadVolumeJSON(r.Body)
		if err != nil {
			t.Errorf("api.ReadVolumeJSON(%v): %v", r, err)
		}

		// We expect the KeyPair to be omitted from the response.
		want := &api.Volume{ID: v.ID, Source: v.Source, Tags: v.Tags}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("http.Post(%v, %v, %v): want %v, got %v", url, cnt, b, want, got)
		}
	})

	t.Run("List", func(t *testing.T) {
		url := fmt.Sprintf("http://%v", addr)

		r, err := http.Get(url)
		if err != nil {
			t.Fatalf("http.Get(%v): %v", url, err)
		}

		if r.StatusCode != http.StatusOK {
			e, _ := ioutil.ReadAll(r.Body)
			t.Fatalf("http.Get(%v): Want %v, got %v: %s", url, http.StatusOK, r.StatusCode, e)
		}

		got, err := api.ReadVolumesJSON(r.Body)
		if err != nil {
			t.Errorf("api.ReadVolumesJSON(%v): %v", r, err)
		}

		want := api.Volumes{&api.Volume{ID: v.ID, Source: v.Source, Tags: v.Tags}}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("http.Get(%v): want %v, got %v", url, want, got)
		}
	})

	t.Run("Get", func(t *testing.T) {
		url := fmt.Sprintf("http://%v/%v", addr, v.ID)

		r, err := http.Get(url)
		if err != nil {
			t.Fatalf("http.Get(%v): %v", url, err)
		}

		if r.StatusCode != http.StatusOK {
			e, _ := ioutil.ReadAll(r.Body)
			t.Fatalf("http.Get(%v): Want %v, got %v: %s", url, http.StatusOK, r.StatusCode, e)
		}

		got, err := api.ReadVolumeJSON(r.Body)
		if err != nil {
			t.Errorf("api.ReadVolumeJSON(%v): %v", r, err)
		}

		want := &api.Volume{ID: v.ID, Source: v.Source, Tags: v.Tags}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("http.Get(%v): want %v, got %v", url, want, got)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		url := fmt.Sprintf("http://%v/%v", addr, v.ID)

		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		c := &http.Client{}
		r, err := c.Do(req)
		if err != nil {
			t.Fatalf("http.Get(%v): %v", url, err)
		}

		if r.StatusCode != http.StatusOK {
			e, _ := ioutil.ReadAll(r.Body)
			t.Fatalf("http.Get(%v): Want %v, got %v: %s", url, http.StatusOK, r.StatusCode, e)
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		url := fmt.Sprintf("http://%v/%v", addr, v.ID)

		r, err := http.Get(url)
		if err != nil {
			t.Fatalf("http.Get(%v): %v", url, err)
		}

		if r.StatusCode != http.StatusNotFound {
			e, _ := ioutil.ReadAll(r.Body)
			t.Fatalf("http.Get(%v): Want %v, got %v: %s", url, http.StatusNotFound, r.StatusCode, e)
		}
	})
}
