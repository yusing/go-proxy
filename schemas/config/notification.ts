import { URL } from "../types";

export const NOTIFICATION_PROVIDERS = ["webhook", "gotify"] as const;

export type NotificationProvider = (typeof NOTIFICATION_PROVIDERS)[number];

export type NotificationConfig = {
  /* Name of the notification provider */
  name: string;
  /* URL of the notification provider */
  url: URL;
};

export interface GotifyConfig extends NotificationConfig {
  provider: "gotify";
  /* Gotify token */
  token: string;
}

export const WEBHOOK_TEMPLATES = ["discord"] as const;
export const WEBHOOK_METHODS = ["POST", "GET", "PUT"] as const;
export const WEBHOOK_MIME_TYPES = [
  "application/json",
  "application/x-www-form-urlencoded",
  "text/plain",
] as const;
export const WEBHOOK_COLOR_MODES = ["hex", "dec"] as const;

export type WebhookTemplate = (typeof WEBHOOK_TEMPLATES)[number];
export type WebhookMethod = (typeof WEBHOOK_METHODS)[number];
export type WebhookMimeType = (typeof WEBHOOK_MIME_TYPES)[number];
export type WebhookColorMode = (typeof WEBHOOK_COLOR_MODES)[number];

export interface WebhookConfig extends NotificationConfig {
  provider: "webhook";
  /**
   * Webhook template
   *
   * @default "discord"
   */
  template?: WebhookTemplate;
  /* Webhook token */
  token?: string;
  /**
   * Webhook message (usally JSON),
   * required when template is not defined
   */
  payload?: string;
  /**
   * Webhook method
   *
   * @default "POST"
   */
  method?: WebhookMethod;
  /**
   * Webhook mime type
   *
   * @default "application/json"
   */
  mime_type?: WebhookMimeType;
  /**
   * Webhook color mode
   *
   * @default "hex"
   */
  color_mode?: WebhookColorMode;
}
