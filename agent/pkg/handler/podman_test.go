package handler_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/agent/pkg/agent"
	"github.com/yusing/godoxy/internal/agentpool"
	"github.com/yusing/godoxy/internal/docker"
	"github.com/yusing/godoxy/internal/types"
	socketproxy "github.com/yusing/godoxy/socketproxy/pkg"
)

type podmanAgentContext struct{ context.Context }

func (c *podmanAgentContext) SetValue(key, value any) {
	c.Context = context.WithValue(c.Context, key, value)
}

func TestPodmanDependenciesUsesAgentForwarding(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "podman.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	appID, dbID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	var inspections atomic.Int32
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.47")
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			fmt.Fprintf(w, `[{"Id":%q,"Names":["/stack_app_1"],"Labels":{"io.podman.compose.project":"stack","com.docker.compose.project":"stack","com.docker.compose.service":"app","proxy.idle_timeout":"1m"}},{"Id":%q,"Names":["/stack_db_1"],"Labels":{"io.podman.compose.project":"stack","com.docker.compose.project":"stack","com.docker.compose.service":"db"}}]`, appID, dbID)
		case r.URL.Path == "/v4.0.0/libpod/containers/"+appID+"/json":
			inspections.Add(1)
			fmt.Fprintf(w, `{"Id":%q,"Dependencies":[%q]}`, appID, dbID)
		case r.URL.Path == "/v4.0.0/libpod/containers/"+dbID+"/json":
			inspections.Add(1)
			fmt.Fprintf(w, `{"Id":%q,"Dependencies":[]}`, dbID)
		default:
			t.Errorf("unexpected Podman request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})}
	go func() { _ = backend.Serve(ln) }()
	t.Cleanup(func() { backend.Close() })

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), DNSNames: []string{agent.AgentHost},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(certPEM))

	mux := http.NewServeMux()
	// Legacy metadata avoids unrelated stream health probes in this HTTP fixture.
	mux.HandleFunc(agent.APIEndpointBase+"/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case agent.APIEndpointBase + agent.EndpointName:
			io.WriteString(w, `test-agent`)
		case agent.APIEndpointBase + agent.EndpointRuntime:
			io.WriteString(w, `podman`)
		case agent.APIEndpointBase + agent.EndpointVersion:
			io.WriteString(w, `v0.0.1`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/", socketproxy.DockerSocketHandler(sock))
	srv := httptest.NewUnstartedServer(mux)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS12}
	srv.StartTLS()
	defer srv.Close()

	cfg := &agent.AgentConfig{Addr: srv.Listener.Addr().String()}
	require.NoError(t, cfg.InitWithCerts(t.Context(), certPEM, certPEM, keyPEM))
	pool := agentpool.NewPool()
	require.True(t, pool.Add(cfg))
	ctx := &podmanAgentContext{t.Context()}
	agentpool.SetCtx(ctx, pool)
	cs, err := docker.ListContainers(ctx, types.DockerProviderConfig{URL: cfg.FakeDockerHost()})
	require.NoError(t, err)
	require.Len(t, cs, 2)
	require.EqualValues(t, 2, inspections.Load())
	require.NotNil(t, cs[0].IdlewatcherConfig)
	require.Equal(t, []string{"db"}, cs[0].IdlewatcherConfig.DependsOn)
	require.Nil(t, cs[1].IdlewatcherConfig)
	require.Empty(t, docker.Dependencies(cs[1]))
}
