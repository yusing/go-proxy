import { URI, URL } from "../types";
import { GotifyConfig, WebhookConfig } from "./notification";

/**
 * @additionalProperties false
 */
export type Providers = {
  /** List of route definition files to include
   *
   * @minItems 1
   * @examples require(".").includeExamples
   * @items.pattern ^[\w\d\-_]+\.(yaml|yml)$
   */
  include?: URI[];
  /** Name-value mapping of docker hosts to retrieve routes from
   *
   * @minProperties 1
   * @examples require(".").dockerExamples
   * @items.pattern ^((\w+://)[^\s]+)|\$DOCKER_HOST$
   */
  docker?: { [name: string]: URL };
  /** List of notification providers
   *
   * @minItems 1
   * @examples require(".").notificationExamples
   */
  notification?: (WebhookConfig | GotifyConfig)[];
};

export const includeExamples = ["file1.yml", "file2.yml"] as const;
export const dockerExamples = [
  { local: "$DOCKER_HOST" },
  { remote: "tcp://10.0.2.1:2375" },
  { remote2: "ssh://root:1234@10.0.2.2" },
] as const;
export const notificationExamples = [
  {
    name: "gotify",
    provider: "gotify",
    url: "https://gotify.domain.tld",
    token: "abcd",
  },
  {
    name: "discord",
    provider: "webhook",
    template: "discord",
    url: "https://discord.com/api/webhooks/1234/abcd",
  },
] as const;
