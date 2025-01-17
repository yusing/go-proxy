import { CIDR, HTTPHeader, HTTPMethod, StatusCodeRange, URI } from "../types";

export const ACCESS_LOG_FORMATS = ["combined", "common", "json"] as const;

export type AccessLogFormat = (typeof ACCESS_LOG_FORMATS)[number];

/**
 * @additionalProperties false
 */
export type AccessLogConfig = {
  /**
   * The size of the buffer.
   *
   * @minimum 0
   * @default 65536
   * @TJS-type integer
   */
  buffer_size?: number;
  /** The format of the access log.
   *
   * @default "combined"
   */
  format?: AccessLogFormat;
  /* The path to the access log file. */
  path: URI;
  /* The access log filters. */
  filters?: AccessLogFilters;
  /* The access log fields. */
  fields?: AccessLogFields;
};

export type AccessLogFilter<T> = {
  /** Whether the filter is negative.
   *
   * @default false
   */
  negative?: boolean;
  /* The values to filter. */
  values: T[];
};

export type AccessLogFilters = {
  /* Status code filter. */
  status_code?: AccessLogFilter<StatusCodeRange>;
  /* Method filter. */
  method?: AccessLogFilter<HTTPMethod>;
  /* Host filter. */
  host?: AccessLogFilter<string>;
  /* Header filter. */
  headers?: AccessLogFilter<HTTPHeader>;
  /* CIDR filter. */
  cidr?: AccessLogFilter<CIDR>;
};

export const ACCESS_LOG_FIELD_MODES = ["keep", "drop", "redact"] as const;
export type AccessLogFieldMode = (typeof ACCESS_LOG_FIELD_MODES)[number];

export type AccessLogField = {
  default?: AccessLogFieldMode;
  config: {
    [key: string]: AccessLogFieldMode;
  };
};

export type AccessLogFields = {
  header?: AccessLogField;
  query?: AccessLogField;
  cookie?: AccessLogField;
};
