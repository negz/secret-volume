package secrets

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
	"github.com/spf13/afero"
)

var mergeTests = []struct {
	f string
	v *api.Volume
	t api.SecretType
	j []byte
}{
	{
		"../fixtures/merge.tar.gz",
		fixtures.TestVolume,
		api.YAMLSecretType,
		[]byte("{\"secret\":\"A\",\"secretA\":\"A\",\"secretB\":\"B\"}\n"),
	},
}

func TestMerge(t *testing.T) {
	fs := afero.NewOsFs()

	for _, tt := range mergeTests {
		s, _ := OpenTarGz(tt.v, fs, tt.f, TarGzSecretType(tt.t))

		t.Run("WriteJSON", func(t *testing.T) {
			b := &bytes.Buffer{}
			if err := WriteJSON(s, b); err != nil {
				t.Errorf("secrets.WriteJSON(%v, %v): %v", s, b, err)
				return
			}
			actual, _ := ioutil.ReadAll(b)
			if !reflect.DeepEqual(actual, tt.j) {
				t.Errorf("ioutil.ReadAll(%v): want %s, got %s", b, tt.j, actual)
			}
		})
	}
}
