import { URL } from "../types";

/**
 * @additionalProperties false
 */
export type HomepageConfig = {
  /** Whether show in dashboard
   *
   * @default true
   */
  show?: boolean;
  /* Display name on dashboard */
  name?: string;
  /* Display icon on dashboard */
  icon?: URL | WalkxcodeIcon | ExternalIcon | TargetRelativeIconPath;
  /* App description */
  description?: string;
  /* Override url */
  url?: URL;
  /* App category */
  category?: string;
  /* Widget config */
  widget_config?: {
    [key: string]: any;
  };
};

/** Walkxcode icon
 *
 * @pattern ^(png|svg|webp)\/[\w\d\-_]+\.\1
 * @type string
 */
export type WalkxcodeIcon = string & {};

/* Walkxcode / selfh.st icon */
export type ExternalIcon = `@${"selfhst" | "walkxcode"}/${string}.${string}`;

/* Relative path to proxy target */
export type TargetRelativeIconPath = `@target/${string}` | `/${string}`;
