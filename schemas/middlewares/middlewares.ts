import * as types from "../types";

export type MiddlewaresMap = {
  redirect_http?: RedirectHTTP;
  redirectHTTP?: RedirectHTTP;
  RedirectHTTP?: RedirectHTTP;
  oidc?: OIDC;
  OIDC?: OIDC;
  request?: ModifyRequest;
  Request?: ModifyRequest;
  modify_request?: ModifyRequest;
  modifyRequest?: ModifyRequest;
  ModifyRequest?: ModifyRequest;
  response?: ModifyResponse;
  Response?: ModifyResponse;
  modify_response?: ModifyResponse;
  modifyResponse?: ModifyResponse;
  ModifyResponse?: ModifyResponse;
  set_x_forwarded?: SetXForwarded;
  setXForwarded?: SetXForwarded;
  SetXForwarded?: SetXForwarded;
  hide_x_forwarded?: HideXForwarded;
  hideXForwarded?: HideXForwarded;
  HideXForwarded?: HideXForwarded;
  error_page?: CustomErrorPage;
  errorPage?: CustomErrorPage;
  custom_error_page?: CustomErrorPage;
  customErrorPage?: CustomErrorPage;
  CustomErrorPage?: CustomErrorPage;
  real_ip?: RealIP;
  realIP?: RealIP;
  RealIP?: RealIP;
  cloudflare_real_ip?: CloudflareRealIP;
  cloudflareRealIP?: CloudflareRealIP;
  CloudflareRealIP?: CloudflareRealIP;
  rate_limit?: RateLimit;
  rateLimit?: RateLimit;
  RateLimit?: RateLimit;
  cidr_whitelist?: CIDRWhitelist;
  cidrWhitelist?: CIDRWhitelist;
  CIDRWhitelist?: CIDRWhitelist;
};

/**
 * @additionalProperties false
 */
export type CustomErrorPage = {};

/**
 * @additionalProperties false
 */
export type CIDRWhitelist = {
  /* Allowed CIDRs/IPs */
  allow: types.CIDR[];
  /** HTTP status code when blocked
   *
   * @default 403
   */
  status_code?: types.StatusCode;
  /** Error message when blocked
   *
   * @default "IP not allowed"
   */
  message?: string;
};

/**
 * @additionalProperties false
 */
export type CloudflareRealIP = {
  /** Recursively resolve the IP
   *
   * @default false
   */
  recursive?: boolean;
};

/**
 * @additionalProperties false
 */
export type ModifyRequest = {
  /** Set HTTP headers */
  set_headers?: { [key: types.HTTPHeader]: string };
  /** Add HTTP headers */
  add_headers?: { [key: types.HTTPHeader]: string };
  /** Hide HTTP headers */
  hide_headers?: types.HTTPHeader[];
};

/**
 * @additionalProperties false
 */
export type ModifyResponse = ModifyRequest;

/**
 * @additionalProperties false
 */
export type OIDC = {
  /** Allowed users
   *
   * @minItems 1
   */
  allowed_users?: string[];
  /** Allowed groups
   *
   * @minItems 1
   */
  allowed_groups?: string[];
};

/**
 * @additionalProperties false
 */
export type RateLimit = {
  /** Average number of requests allowed in a period
   *
   * @min 1
   */
  average: number;
  /** Maximum number of requests allowed in a period
   *
   * @min 1
   */
  burst: number;
  /** Duration of the rate limit
   *
   * @default 1s
   */
  period?: types.Duration;
};

/**
 * @additionalProperties false
 */
export type RealIP = {
  /** Header to get the client IP from
   *
   */
  header: types.HTTPHeader;
  from: types.CIDR[];
  /** Recursive resolve the IP
   *
   * @default false
   */
  recursive: boolean;
};

/**
 * @additionalProperties false
 */
export type RedirectHTTP = {};

/**
 * @additionalProperties false
 */
export type SetXForwarded = {};

/**
 * @additionalProperties false
 */
export type HideXForwarded = {};
