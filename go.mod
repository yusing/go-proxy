module github.com/yusing/go-proxy

go 1.23.6

require (
	github.com/PuerkitoBio/goquery v1.10.2 // parsing HTML for extract fav icon
	github.com/coder/websocket v1.8.12 // websocket for API and agent
	github.com/coreos/go-oidc/v3 v3.12.0 // oidc authentication
	github.com/docker/cli v27.5.1+incompatible // docker CLI
	github.com/docker/docker v27.5.1+incompatible // docker daemon
	github.com/fsnotify/fsnotify v1.8.0 // file watcher
	github.com/go-acme/lego/v4 v4.22.2 // acme client
	github.com/go-playground/validator/v10 v10.25.0 // validator
	github.com/gobwas/glob v0.2.3 // glob matcher for route rules
	github.com/golang-jwt/jwt/v5 v5.2.1 // jwt for default auth
	github.com/gotify/server/v2 v2.6.1 // reference the Message struct for json response
	github.com/lithammer/fuzzysearch v1.1.8 // fuzzy search for searching icons and filtering metrics
	github.com/prometheus/client_golang v1.20.5 // metrics
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // lock free map for concurrent operations
	github.com/rs/zerolog v1.33.0 // logging
	github.com/vincent-petithory/dataurl v1.0.0 // data url for fav icon
	golang.org/x/crypto v0.33.0 // encrypting password with bcrypt
	golang.org/x/net v0.35.0 // HTTP header utilities
	golang.org/x/oauth2 v0.26.0 // oauth2 authentication
	golang.org/x/text v0.22.0 // string utilities
	golang.org/x/time v0.10.0 // time utilities
	gopkg.in/yaml.v3 v3.0.1 // yaml parsing for different config files
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/cloudflare-go v0.115.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.63 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nrdcg/porkbun v0.4.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/ovh/go-ovh v1.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.30.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.30.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)
