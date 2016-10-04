package secrets

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"

	"github.com/negz/secret-volume/api"

	"github.com/benschw/srv-lb/lb"
	"github.com/uber-go/zap"
)

type talosSecretProducer struct {
	lb  lb.LoadBalancer
	ctx context.Context
}

type TalosSecretProviderOption func(sp *talosSecretProducer) error

func WithContext(ctx context.Context) TalosSecretProviderOption {
	return func(sp *talosSecretProducer) error {
		sp.ctx = ctx
		return nil
	}
}

func NewTalosSecretProducer(lb lb.LoadBalancer, spo ...TalosSecretProviderOption) (SecretProducer, error) {
	sp := &talosSecretProducer{lb, context.Background()}
	for _, o := range spo {
		if err := o(sp); err != nil {
			return nil, err
		}
	}
	return sp, nil
}

func httpClientFor(v *api.Volume) (*http.Client, error) {
	crt, err := v.KeyPair.ToCertificate()
	if err != nil {
		return nil, err
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{crt}, InsecureSkipVerify: true}
	cfg.BuildNameToCertificate()

	return &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}, nil
}

func (sp *talosSecretProducer) url(tags url.Values) (string, error) {
	h, err := sp.lb.Next()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%v?%v", h, tags.Encode()), nil
}
func (sp *talosSecretProducer) For(v *api.Volume) (api.Secrets, error) {
	url, err := sp.url(v.Tags)
	if err != nil {
		return nil, err
	}
	log.Debug("fetching secrets", zap.String("url", url))
	ctx, cancel := context.WithTimeout(sp.ctx, 15*time.Second)
	defer cancel()
	c, err := httpClientFor(v)
	if err != nil {
		return nil, err
	}
	r, err := ctxhttp.Get(ctx, c, url)
	if err != nil {
		return nil, err
	}
	return NewTarGzSecrets(v, r.Body)
}
