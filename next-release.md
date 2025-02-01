GoDoxy v0.9.1 expected changes

- Support Ntfy notifications
- Prometheus metrics server now inside API server under `/v1/metrics`
  - `GODOXY_PROMETHEUS_ADDR` removed
  - `GODOXY_PROMETHEUS_ENABLED` added, default `false`
