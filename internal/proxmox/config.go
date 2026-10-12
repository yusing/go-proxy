package proxmox

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/luthermonson/go-proxmox"
	"github.com/rs/zerolog/log"
	strutils "github.com/yusing/goutils/strings"
)

type Config struct {
	URL string `json:"url" validate:"required,url"`

	Username string            `json:"username" validate:"required_without_all=TokenID Secret"`
	Password strutils.Redacted `json:"password" validate:"required_without_all=TokenID Secret"`
	Realm    string            `json:"realm"` // default is "pam"

	TokenID string            `json:"token_id" validate:"required_without_all=Username Password"`
	Secret  strutils.Redacted `json:"secret" validate:"required_without_all=Username Password"`

	NoTLSVerify bool `json:"no_tls_verify" yaml:"no_tls_verify,omitempty"`

	client *Client
}

const (
	ResourcePollInterval   = 3 * time.Second
	SessionRefreshInterval = 1 * time.Minute
	RequestTimeout         = 30 * time.Second

	maxConcurrentResourceLookups = 3
	proxmoxConnectionHeadroom    = 3
	proxmoxMaxConnsPerHost       = maxConcurrentResourceLookups + proxmoxConnectionHeadroom
)

// NodeStatsPollInterval controls how often node stats are streamed when streaming is enabled.
const NodeStatsPollInterval = time.Second

func (c *Config) Client() *Client {
	if c.client == nil {
		panic("proxmox client accessed before init")
	}
	return c.client
}

func newProxmoxHTTPClient(noTLSVerify bool) *http.Client {
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          proxmoxMaxConnsPerHost,
		MaxIdleConnsPerHost:   proxmoxMaxConnsPerHost,
		MaxConnsPerHost:       proxmoxMaxConnsPerHost,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: RequestTimeout,
		IdleConnTimeout:       90 * time.Second,
	}
	if noTLSVerify {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		}
	}
	return &http.Client{
		Transport: tr,
		Timeout:   RequestTimeout,
	}
}

func (c *Config) Init(ctx context.Context) error {
	c.URL = strings.TrimSuffix(c.URL, "/")
	if !strings.HasSuffix(c.URL, "/api2/json") {
		c.URL += "/api2/json"
	}

	opts := []proxmox.Option{
		proxmox.WithHTTPClient(newProxmoxHTTPClient(c.NoTLSVerify)),
	}
	useCredentials := false
	if c.Username != "" && c.Password != "" {
		if c.Realm == "" {
			c.Realm = "pam"
		}
		opts = append(opts, proxmox.WithCredentials(&proxmox.Credentials{
			Username: c.Username,
			Password: c.Password.String(),
			Realm:    c.Realm,
		}))
		useCredentials = true
	} else {
		opts = append(opts, proxmox.WithAPIToken(c.TokenID, c.Secret.String()))
	}
	c.client = NewClient(c.URL, opts...)

	initCtx, initCtxCancel := context.WithTimeout(ctx, RequestTimeout)
	defer initCtxCancel()

	authenticated := !useCredentials
	if err := retryInitialization(initCtx, func() error {
		if !authenticated {
			if err := c.client.CreateSession(initCtx); err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}
			authenticated = true
		}
		if err := c.client.UpdateClusterInfo(initCtx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout fetching proxmox cluster info: %w", err)
			}
			return fmt.Errorf("failed to fetch proxmox cluster info: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	{
		reqCtx, reqCtxCancel := context.WithTimeout(ctx, RequestTimeout)
		err := c.client.UpdateResources(reqCtx)
		reqCtxCancel()
		if err != nil {
			log.Warn().Err(err).Str("cluster", c.client.Cluster.Name).Msg("[proxmox] failed to update resources")
		}
	}

	go c.updateResourcesLoop(ctx)
	go c.refreshSessionLoop(ctx)
	return nil
}

// Retry only bootstrap connections, sharing the existing initialization deadline.
// LXC actions must not be replayed after an ambiguous transport failure.
func retryInitialization(ctx context.Context, initialize func() error) error {
	delay := 250 * time.Millisecond
	for {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		err := initialize()
		if err == nil {
			return nil
		}
		if cause := context.Cause(ctx); cause != nil {
			return errors.Join(err, cause)
		}
		var networkError *net.OpError
		var dnsError *net.DNSError
		if !errors.As(err, &networkError) && !errors.As(err, &dnsError) &&
			!errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(err, context.Cause(ctx))
		case <-timer.C:
		}
		delay = min(delay*2, 3*time.Second)
	}
}

func (c *Config) refreshSession(ctx context.Context) error {
	if _, err := c.client.Client.Version(ctx); err == nil {
		return nil
	}
	if err := c.client.RefreshTicket(ctx); err == nil {
		return nil
	}
	// CreateSession refuses to replace a locally unexpired session, so force
	// full reauthentication through the public ticket endpoint.
	_, err := c.client.Ticket(ctx, &proxmox.Credentials{
		Username: c.Username,
		Password: c.Password.String(),
		Realm:    c.Realm,
	})
	return err
}

func (c *Config) updateResourcesLoop(ctx context.Context) {
	ticker := time.NewTicker(ResourcePollInterval)
	defer ticker.Stop()

	log.Trace().Str("cluster", c.client.Cluster.Name).Msg("[proxmox] starting resources update loop")

	for {
		select {
		case <-ctx.Done():
			log.Trace().Str("cluster", c.client.Cluster.Name).Msg("[proxmox] stopping resources update loop")
			return
		case <-ticker.C:
			reqCtx, reqCtxCancel := context.WithTimeout(ctx, RequestTimeout)
			err := c.client.UpdateResources(reqCtx)
			reqCtxCancel()
			if err != nil {
				log.Error().Err(err).Str("cluster", c.client.Cluster.Name).Msg("[proxmox] failed to update resources")
			}
			ticker.Reset(ResourcePollInterval)
		}
	}
}

func (c *Config) refreshSessionLoop(ctx context.Context) {
	ticker := time.NewTicker(SessionRefreshInterval)
	defer ticker.Stop()

	log.Trace().Str("cluster", c.client.Cluster.Name).Msg("[proxmox] starting session refresh loop")

	numRetries := 0
	for {
		select {
		case <-ctx.Done():
			log.Trace().Str("cluster", c.client.Cluster.Name).Msg("[proxmox] stopping session refresh loop")
			return
		case <-ticker.C:
			reqCtx, reqCtxCancel := context.WithTimeout(ctx, RequestTimeout)
			err := c.refreshSession(reqCtx)
			reqCtxCancel()
			if err != nil {
				log.Err(err).Str("cluster", c.client.Cluster.Name).Msg("[proxmox] failed to refresh session")
				numRetries++
				backoff := 10 * time.Second
				if numRetries <= 3 {
					backoff = time.Duration(1<<numRetries) * time.Second
				}
				ticker.Reset(backoff)
			} else {
				numRetries = 0
				ticker.Reset(SessionRefreshInterval)
			}
		}
	}
}
