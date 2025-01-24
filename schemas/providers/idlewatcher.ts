import { Duration, URI } from "../types";

export const STOP_METHODS = ["pause", "stop", "kill"] as const;
export type StopMethod = (typeof STOP_METHODS)[number];

export const STOP_SIGNALS = [
  "",
  "SIGINT",
  "SIGTERM",
  "SIGHUP",
  "SIGQUIT",
  "INT",
  "TERM",
  "HUP",
  "QUIT",
] as const;
export type Signal = (typeof STOP_SIGNALS)[number];

export type IdleWatcherConfig = {
  /* Idle timeout */
  idle_timeout?: Duration;
  /** Wake timeout
   *
   * @default 30s
   */
  wake_timeout?: Duration;
  /** Stop timeout
   *
   * @default 30s
   */
  stop_timeout?: Duration;
  /** Stop method
   *
   * @default stop
   */
  stop_method?: StopMethod;
  /* Stop signal */
  stop_signal?: Signal;
  /* Start endpoint (any path can wake the container if not specified) */
  start_endpoint?: URI;
};
