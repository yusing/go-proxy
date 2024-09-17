# Supported DNS Providers

<!-- TOC -->

- [Supported DNS Providers](#supported-dns-providers)
  - [Cloudflare](#cloudflare)
  - [CloudDNS](#clouddns)
  - [DuckDNS](#duckdns)
  - [OVHCloud](#ovhcloud)
  - [Implement other DNS providers](#implement-other-dns-providers)

## Cloudflare

`auth_token` your zone API token

Follow [this guide](https://cloudkul.com/blog/automcatic-renew-and-generate-ssl-on-your-website-using-lego-client/) to create a new token with `Zone.DNS` read and edit permissions

## CloudDNS

- `client_id`

- `email`

- `password`

## DuckDNS

- `token`: DuckDNS Token

Tested by [earvingad](https://github.com/earvingad)

## OVHCloud

_Note, `application_key` and `oauth2_config` **CANNOT** be used together_

- `api_endpoint`: Endpoint URL, or one of
  - `ovh-eu`,
  - `ovh-ca`,
  - `ovh-us`,
  - `kimsufi-eu`,
  - `kimsufi-ca`,
  - `soyoustart-eu`,
  - `soyoustart-ca`
- `application_secret`
- `application_key`
- `consumer_key`
- `oauth2_config`: Client ID and Client Secret
  - `client_id`
  - `client_secret`

## Implement other DNS providers

See [add_dns_provider.md](docs/add_dns_provider.md)
