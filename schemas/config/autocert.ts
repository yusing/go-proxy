import { DomainOrWildcard, Email } from "../types";

export const AUTOCERT_PROVIDERS = [
  "local",
  "cloudflare",
  "clouddns",
  "duckdns",
  "ovh",
] as const;

export type AutocertProvider = (typeof AUTOCERT_PROVIDERS)[number];

export type AutocertConfig =
  | LocalOptions
  | CloudflareOptions
  | CloudDNSOptions
  | DuckDNSOptions
  | OVHOptionsWithAppKey
  | OVHOptionsWithOAuth2Config;

export interface AutocertConfigBase {
  /* ACME email */
  email: Email;
  /* ACME domains */
  domains: DomainOrWildcard[];
  /* ACME certificate path */
  cert_path?: string;
  /* ACME key path */
  key_path?: string;
}

export interface LocalOptions {
  provider: "local";
  /* ACME certificate path */
  cert_path?: string;
  /* ACME key path */
  key_path?: string;
  options?: {} | null;
}

export interface CloudflareOptions extends AutocertConfigBase {
  provider: "cloudflare";
  options: { auth_token: string };
}

export interface CloudDNSOptions extends AutocertConfigBase {
  provider: "clouddns";
  options: {
    client_id: string;
    email: Email;
    password: string;
  };
}

export interface DuckDNSOptions extends AutocertConfigBase {
  provider: "duckdns";
  options: {
    token: string;
  };
}

export const OVH_ENDPOINTS = [
  "ovh-eu",
  "ovh-ca",
  "ovh-us",
  "kimsufi-eu",
  "kimsufi-ca",
  "soyoustart-eu",
  "soyoustart-ca",
] as const;

export type OVHEndpoint = (typeof OVH_ENDPOINTS)[number];

export interface OVHOptionsWithAppKey extends AutocertConfigBase {
  provider: "ovh";
  options: {
    application_secret: string;
    consumer_key: string;
    api_endpoint?: OVHEndpoint;
    application_key: string;
  };
}

export interface OVHOptionsWithOAuth2Config extends AutocertConfigBase {
  provider: "ovh";
  options: {
    application_secret: string;
    consumer_key: string;
    api_endpoint?: OVHEndpoint;
    oauth2_config: {
      client_id: string;
      client_secret: string;
    };
  };
}
