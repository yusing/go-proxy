import { URL } from "../types";
export declare const NOTIFICATION_PROVIDERS: readonly ["webhook", "gotify"];
export type NotificationProvider = (typeof NOTIFICATION_PROVIDERS)[number];
export type NotificationConfig = {
    name: string;
    url: URL;
};
export interface GotifyConfig extends NotificationConfig {
    provider: "gotify";
    token: string;
}
export declare const WEBHOOK_TEMPLATES: readonly ["", "discord"];
export declare const WEBHOOK_METHODS: readonly ["POST", "GET", "PUT"];
export declare const WEBHOOK_MIME_TYPES: readonly ["application/json", "application/x-www-form-urlencoded", "text/plain"];
export declare const WEBHOOK_COLOR_MODES: readonly ["hex", "dec"];
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
