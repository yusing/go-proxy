import { MiddlewareCompose } from "../middlewares/middleware_compose";
import { AccessLogConfig } from "./access_log";

export type EntrypointConfig = {
  /** Entrypoint middleware configuration
   *
   * @examples require(".").middlewaresExamples
   */
  middlewares: MiddlewareCompose;
  /** Entrypoint access log configuration
   *
   * @examples require(".").accessLogExamples
   */
  access_log?: AccessLogConfig;
};

export const accessLogExamples = [
  {
    path: "/var/log/access.log",
    format: "combined",
    filters: {
      status_codes: {
        values: ["200-299"],
      },
    },
    fields: {
      headers: {
        default: "keep",
        config: {
          foo: "redact",
        },
      },
    },
  },
] as const;

export const middlewaresExamples = [
  {
    use: "RedirectHTTP",
  },
  {
    use: "CIDRWhitelist",
    allow: ["127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"],
    status: 403,
    message: "Forbidden",
  },
] as const;
