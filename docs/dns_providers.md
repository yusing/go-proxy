# Supported DNS Providers

<!-- TOC -->
- [Cloudflare](#cloudflare)
- [CloudDNS](#clouddns)
- [DuckDNS](#duckdns)
- [Implement other DNS providers](#implement-other-dns-providers)
<!-- /TOC -->

## Cloudflare

`auth_token` your zone API token

Follow [this guide](https://cloudkul.com/blog/automcatic-renew-and-generate-ssl-on-your-website-using-lego-client/) to create a new token with `Zone.DNS` read and edit permissions

## CloudDNS

- `client_id`

- `email`

- `password`

## DuckDNS

`token`: DuckDNS Token

Tested by [earvingad](https://github.com/earvingad)

## Implement other DNS providers

See [add_dns_provider.md](docs/add_dns_provider.md)
