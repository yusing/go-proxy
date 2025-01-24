import { URI, URL } from "../types";
import { GotifyConfig, WebhookConfig } from "./notification";
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
     */
    docker?: {
        [name: string]: URL | "$DOCKER_HOST";
    };
    /** List of notification providers
     *
     * @minItems 1
     * @examples require(".").notificationExamples
     */
    notification?: (WebhookConfig | GotifyConfig)[];
};
export declare const includeExamples: readonly ["file1.yml", "file2.yml"];
export declare const dockerExamples: readonly [{
    readonly local: "$DOCKER_HOST";
}, {
    readonly remote: "tcp://10.0.2.1:2375";
}, {
    readonly remote2: "ssh://root:1234@10.0.2.2";
}];
export declare const notificationExamples: readonly [{
    readonly name: "gotify";
    readonly provider: "gotify";
    readonly url: "https://gotify.domain.tld";
    readonly token: "abcd";
}, {
    readonly name: "discord";
    readonly provider: "webhook";
    readonly template: "discord";
    readonly url: "https://discord.com/api/webhooks/1234/abcd";
}];
