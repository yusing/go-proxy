# Supported DNS Providers

<!-- TOC -->

- [Supported DNS Providers](#supported-dns-providers)
  - [Cloudflare](#cloudflare)
  - [CloudDNS](#clouddns)
  - [DuckDNS](#duckdns)
  - [OVHCloud](#ovhcloud)
  - [Implement other DNS providers](#implement-other-dns-providers)

## Cloudflare

```yaml
autocert:
    provider: cloudflare
    options:
        auth_token:
```

`auth_token` your zone API token

Follow [this guide](https://cloudkul.com/blog/automcatic-renew-and-generate-ssl-on-your-website-using-lego-client/) to create a new token with `Zone.DNS` read and edit permissions

## CloudDNS

```yaml
autocert:
    provider: clouddns
    options:
        client_id:
        email:
        password:
```

## DuckDNS

```yaml
autocert:
    provider: duckdns
    options:
        token:
```

Tested by [earvingad](https://github.com/earvingad)

## OVHCloud

```yaml
autocert:
    provider: ovh
    options:
        api_endpoint:
        application_key:
        application_secret:
        consumer_key:
        oauth2_config:
            client_id:
            client_secret:
```

_Note, `application_key` and `oauth2_config` **CANNOT** be used together_

-   `api_endpoint`: Endpoint URL, or one of
    -   `ovh-eu`,
    -   `ovh-ca`,
    -   `ovh-us`,
    -   `kimsufi-eu`,
    -   `kimsufi-ca`,
    -   `soyoustart-eu`,
    -   `soyoustart-ca`
-   `application_secret`
-   `application_key`
-   `consumer_key`
-   `oauth2_config`: Client ID and Client Secret
    -   `client_id`
    -   `client_secret`

## Implement other DNS providers

See [add_dns_provider.md](docs/add_dns_provider.md)
