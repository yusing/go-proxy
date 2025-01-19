import { RealIP } from "../middlewares/middlewares";

export const LOAD_BALANCE_MODES = [
  "round_robin",
  "least_conn",
  "ip_hash",
] as const;
export type LoadBalanceMode = (typeof LOAD_BALANCE_MODES)[number];

export type LoadBalanceConfigBase = {
  /** Alias (subdomain or FDN) of load-balancer
   *
   * @minLength 1
   */
  link: string;
  /** Load-balance weight (reserved for future use)
   *
   * @minimum 0
   * @maximum 100
   */
  weight?: number;
};

export type LoadBalanceConfig = LoadBalanceConfigBase &
  (
    | {} // linking other routes
    | RoundRobinLoadBalanceConfig
    | LeastConnLoadBalanceConfig
    | IPHashLoadBalanceConfig
  );

export type IPHashLoadBalanceConfig = {
  mode: "ip_hash";
  /** Real IP config, header to get client IP from */
  config: RealIP;
};

export type LeastConnLoadBalanceConfig = {
  mode: "least_conn";
};

export type RoundRobinLoadBalanceConfig = {
  mode: "round_robin";
};
