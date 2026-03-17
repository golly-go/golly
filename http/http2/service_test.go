package http2_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	golly "github.com/golly-go/golly"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	svchttp2 "github.com/golly-go/golly/http/http2"
)

func TestHTTP2Service_Name(t *testing.T) {
	assert.Equal(t, "http2", svchttp2.New().Name())
}

func TestHTTP2Service_IsRunning(t *testing.T) {
	assert.False(t, svchttp2.New().IsRunning())
}

func TestHTTP2Service_Initialize(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	t.Run("returns error without cert/key", func(t *testing.T) {
		svc := svchttp2.New()
		assert.ErrorIs(t, svc.Initialize(app), svchttp2.ErrTLSConfigRequired)
	})

	t.Run("defaults to :443 with cert/key set", func(t *testing.T) {
		svc := svchttp2.New()
		svc.Configure(func(_ *golly.Application) (svchttp2.Options, error) {
			return svchttp2.Options{CertFile: "cert.pem", KeyFile: "key.pem"}, nil
		})
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":443", svc.Addr())
	})

	t.Run("ConfigFunc overrides bind", func(t *testing.T) {
		svc := svchttp2.New()
		svc.Configure(func(_ *golly.Application) (svchttp2.Options, error) {
			return svchttp2.Options{Bind: ":8443", CertFile: "cert.pem", KeyFile: "key.pem"}, nil
		})
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":8443", svc.Addr())
	})

	t.Run("reads cert/key from app config", func(t *testing.T) {
		app.Config().Set("tls.cert", "from-config.pem")
		app.Config().Set("tls.key", "from-config-key.pem")
		defer func() {
			app.Config().Set("tls.cert", "")
			app.Config().Set("tls.key", "")
		}()
		svc := svchttp2.New()
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":443", svc.Addr())
	})
}

func TestHTTP2Service_StopNotRunning(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	cert, key := generateSelfSignedCert(t)

	svc := svchttp2.New()
	svc.Configure(func(_ *golly.Application) (svchttp2.Options, error) {
		return svchttp2.Options{CertFile: cert, KeyFile: key}, nil
	})
	require.NoError(t, svc.Initialize(app))
	assert.NoError(t, svc.Stop())
}

// TestHTTP2Service_StartStop spins up a real TLS server with a self-signed cert,
// verifies HTTP/2 is negotiated via ALPN, then shuts it down.
func TestHTTP2Service_StartStop(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	app.Routes().Get("/ping", func(wctx *golly.WebContext) {
		wctx.RenderText("pong")
	})

	certFile, keyFile := generateSelfSignedCert(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	svc := svchttp2.New()
	svc.Configure(func(_ *golly.Application) (svchttp2.Options, error) {
		return svchttp2.Options{Bind: addr, CertFile: certFile, KeyFile: keyFile}, nil
	})
	require.NoError(t, svc.Initialize(app))

	startErr := make(chan error, 1)
	go func() { startErr <- svc.Start() }()

	require.Eventually(t, svc.IsRunning, 2*time.Second, 10*time.Millisecond)

	certPEM, err := os.ReadFile(certFile)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certPEM)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{RootCAs: pool},
			ForceAttemptHTTP2: true,
		},
	}

	resp, err := client.Get(fmt.Sprintf("https://%s/ping", addr))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "HTTP/2.0", resp.Proto, "expected HTTP/2 via ALPN")

	require.NoError(t, svc.Stop())

	select {
	case err := <-startErr:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func generateSelfSignedCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	cf, err := os.CreateTemp(t.TempDir(), "cert*.pem")
	require.NoError(t, err)
	require.NoError(t, pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	cf.Close()

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	kf, err := os.CreateTemp(t.TempDir(), "key*.pem")
	require.NoError(t, err)
	require.NoError(t, pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	kf.Close()

	return cf.Name(), kf.Name()
}
