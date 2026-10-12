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
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
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
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v4.0.0/libpod/containers/"+appID+"/json", r.URL.Path)
		fmt.Fprintf(w, `{"Id":%q,"Dependencies":[%q]}`, appID, dbID)
	})}
	go backend.Serve(ln)
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
	snapshot := []container.Summary{{ID: dbID, Names: []string{"/db"}}}
	providerCtx := docker.WithPodmanContainers(ctx, snapshot)
	c := docker.FromDocker(providerCtx, &container.Summary{
		ID: appID, Names: []string{"/app"},
		Labels: map[string]string{"io.podman.compose.project": "repro", docker.LabelIdleTimeout: "10s"},
	}, types.DockerProviderConfig{URL: agent.FakeDockerHostPrefix + cfg.Addr})
	require.Nil(t, c.Errors)
	require.NotNil(t, c.IdlewatcherConfig)
	require.Equal(t, []string{"db"}, c.IdlewatcherConfig.DependsOn)
}
