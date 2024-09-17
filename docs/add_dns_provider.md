# Adding provider support

## **CloudDNS** as an example

1. Fork this repo, modify [autocert.go](../src/go-proxy/autocert.go#L305)

   ```go
   var providersGenMap = map[string]ProviderGenerator{
     "cloudflare": providerGenerator(cloudflare.NewDefaultConfig, cloudflare.NewDNSProviderConfig),
     // add here, e.g.
     "clouddns": providerGenerator(clouddns.NewDefaultConfig, clouddns.NewDNSProviderConfig),
   }
   ```

2. Go to [https://go-acme.github.io/lego/dns/clouddns](https://go-acme.github.io/lego/dns/clouddns/) and check for required config

3. Build `go-proxy` with `make build`

4. Set required config in `config.yml` `autocert` -> `options` section

   ```shell
   # From https://go-acme.github.io/lego/dns/clouddns/
   CLOUDDNS_CLIENT_ID=bLsdFAks23429841238feb177a572aX \
   CLOUDDNS_EMAIL=you@example.com \
   CLOUDDNS_PASSWORD=b9841238feb177a84330f \
   lego --email you@example.com --dns clouddns --domains my.example.org run
   ```

   Should turn into:

   ```yaml
   autocert:
     ...
     options:
       client_id: bLsdFAks23429841238feb177a572aX
       email: you@example.com
       password: b9841238feb177a84330f
   ```

5. Run with `GOPROXY_NO_SCHEMA_VALIDATION=1` and test if it works
6. Commit and create pull request
