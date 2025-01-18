/**
 * @type "null"
 */
export interface Null {}
export type Nullable<T> = T | Null;
export type NullOrEmptyMap = {} | Null;

export const HTTP_METHODS = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "CONNECT",
  "HEAD",
  "OPTIONS",
  "TRACE",
] as const;

export type HTTPMethod = (typeof HTTP_METHODS)[number];
/**
 * HTTP Header
 * @pattern ^[a-zA-Z0-9\-]+$
 */
export type HTTPHeader = string;

/**
 * HTTP Query
 * @pattern ^[a-zA-Z0-9\-_]+$
 */
export type HTTPQuery = string;
/**
 * HTTP Cookie
 * @pattern ^[a-zA-Z0-9\-_]+$
 */
export type HTTPCookie = string;

export type StatusCode = number | `${number}`;
export type StatusCodeRange = number | `${number}` | `${number}-${number}`;

/**
 * @items.pattern ^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$
 */
export type DomainNames = string[];
/**
 * @items.pattern ^(\*\.)?(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$
 */
export type DomainOrWildcards = string[];
/**
 * @format hostname
 */
export type Hostname = string;
/**
 * @format ipv4
 */
export type IPv4 = string;
/**
 * @format ipv6
 */
export type IPv6 = string;

/* CIDR / IPv4 / IPv6 */
export type CIDR =
  | `${number}.${number}.${number}.${number}`
  | `${string}:${string}:${string}:${string}:${string}:${string}:${string}:${string}`
  | `${number}.${number}.${number}.${number}/${number}`
  | `::${number}`
  | `${string}::/${number}`
  | `${string}:${string}::/${number}`;

/**
 * @type integer
 * @minimum 0
 * @maximum 65535
 */
export type Port = number;

/**
 * @pattern ^\d+:\d+$
 */
export type StreamPort = string;

/**
 * @format email
 */
export type Email = string;

/**
 * @format uri
 */
export type URL = string;

/**
 * @format uri-reference
 */
export type URI = string;

/**
 * @pattern ^(?:([A-Z]+) )?(?:([a-zA-Z0-9.-]+)\\/)?(\\/[^\\s]*)$
 */
export type PathPattern = string;

/**
 * @pattern ^([0-9]+(ms|s|m|h))+$
 */
export type Duration = string;

/**
 * @format date-time
 */
export type DateTime = string;
