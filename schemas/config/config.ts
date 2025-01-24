import { DomainName } from "../types";
import { AutocertConfig } from "./autocert";
import { EntrypointConfig } from "./entrypoint";
import { HomepageConfig } from "./homepage";
import { Providers } from "./providers";

export type Config = {
  /** Optional autocert configuration
   *
   * @examples require(".").autocertExamples
   */
  autocert?: AutocertConfig;
  /* Optional entrypoint configuration */
  entrypoint?: EntrypointConfig;
  /* Providers configuration (include file, docker, notification) */
  providers: Providers;
  /** Optional list of domains to match
   *
   * @minItems 1
   * @examples require(".").matchDomainsExamples
   */
  match_domains?: DomainName[];
  /* Optional homepage configuration */
  homepage?: HomepageConfig;
  /**
   * Optional timeout before shutdown
   * @default 3
   * @minimum 1
   */
  timeout_shutdown?: number;
};

export const autocertExamples = [
  { provider: "local" },
  {
    provider: "cloudflare",
    email: "abc@gmail",
    domains: ["example.com"],
    options: { auth_token: "c1234565789-abcdefghijklmnopqrst" },
  },
  {
    provider: "clouddns",
    email: "abc@gmail",
    domains: ["example.com"],
    options: {
      client_id: "c1234565789",
      email: "abc@gmail",
      password: "password",
    },
  },
];
export const matchDomainsExamples = ["example.com", "*.example.com"] as const;
