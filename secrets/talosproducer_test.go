package secrets

import (
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/negz/secret-volume/fixtures"

	"github.com/spf13/afero"
)

var talosSecretsProducerTests = []struct {
	c string
	k string
	f string
}{
	{
		"../fixtures/cert.pem",
		"../fixtures/key.pem",
		"../fixtures/yaml.tar.gz",
	},
}

func TestTalosProducer(t *testing.T) {
	for _, tt := range talosSecretsProducerTests {
		v, _ := fixtures.TestVolumeWithCert(tt.c, tt.k)

		fs := afero.NewOsFs()
		expected, _ := OpenTarGz(v, fs, tt.f)
		defer expected.Close()

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			z, err := fs.Open(tt.f)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			io.Copy(w, z)
			z.Close()
		}))
		ts.StartTLS()
		defer ts.Close()

		lb, err := fixtures.PredictableLoadBalancerFor(ts.URL)
		if err != nil {
			t.Errorf("fixtures.PredictableLoadBalancerFor(%v): %v", ts.URL, err)
			continue
		}

		sp, err := NewTalosProducer(lb)
		if err != nil {
			t.Errorf("NewTalosProducer(%v): %v", lb, err)
			continue
		}

		t.Run("For", func(t *testing.T) {
			actual, err := sp.For(v)
			if err != nil {
				t.Errorf("sp.For(%v): %v", v, err)
				return
			}
			defer actual.Close()
			if actual.Volume() != expected.Volume() {
				t.Errorf("wanted %v, got %v", actual, expected)
			}
			for {
				eh, ee := expected.Next()
				ah, ae := actual.Next()

				if ae != ee {
					// The actual iterator returned a different error than the
					// expected iterator. This could indicate one iterator
					// contained less files than the other.
					t.Errorf("actual.Next(): wanted %v, got %v", ee, ae)
					return
				}

				if ae == io.EOF {
					// Both iterators are at EOF. All is well with the world.
					return
				}

				if ae != nil {
					// Both iterators raised the same, non-EOF error.
					t.Errorf("actual.Next() and expected.Next(): %v", ae)
					return
				}

				if ah.Path != eh.Path {
					// The actual iterator returned a different header than the
					// expected one.
					t.Errorf("actual.Next().Path: wanted %v, got %v", eh, ah)
					return
				}

				efnv := fnv.New64a()
				afnv := fnv.New64a()
				io.Copy(efnv, expected)
				io.Copy(afnv, actual)
				esum := efnv.Sum64()
				asum := efnv.Sum64()
				if esum != asum {
					t.Errorf("%v: wanted hash %v, got %v", ah.Path, fmt.Sprintf("%016x", esum), fmt.Sprintf("%016x", asum))
				}
			}
		})
	}
}
