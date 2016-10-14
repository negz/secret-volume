package secrets

import (
	"bytes"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
)

var singleSecretTests = []struct {
	v *api.Volume
	n string
	b []byte
	t api.SecretType
}{
	{
		fixtures.TestVolume,
		"fixture",
		[]byte("bytes"),
		api.JSONSecretType,
	},
}

func TestSingleSecrets(t *testing.T) {
	for _, tt := range singleSecretTests {
		r := bytes.NewBuffer(tt.b)
		ss := NewSingleFile(tt.v, tt.n, r, tt.t)

		t.Run("Volume", func(t *testing.T) {
			if ss.Volume() != tt.v {
				t.Errorf("ss.Volume(), want %v, got %v", tt.v, ss.Volume())
			}
		})
		t.Run("Next", func(t *testing.T) {
			h, err := ss.Next()
			if err != nil {
				t.Errorf("ss.Next(): %v", err)
				return
			}
			if h.Path != tt.n {
				t.Errorf("h.Path, want %v, got %v", tt.n, h.Path)
			}
			if h.FileInfo.Name() != tt.n {
				t.Errorf("h.FileInfo.Name(), want %v, got %v", tt.n, h.FileInfo.Name())
			}
			_, err = ss.Next()
			if err != io.EOF {
				t.Errorf("ss.Next(): want %v, got %v", io.EOF, err)
			}
		})
		t.Run("Read", func(t *testing.T) {
			r, err := ioutil.ReadAll(ss)
			if err != nil {
				t.Errorf("ioutil.ReadAll(%v): %v", ss, err)
			}
			if !reflect.DeepEqual(r, tt.b) {
				t.Errorf("ioutil.ReadAll(%v): want %v, got %v", ss, tt.b, r)
			}
		})
	}
}
