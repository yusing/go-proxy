import { URL } from "../types";

/**
 * @additionalProperties false
 */
export type HomepageConfig = {
  /* Display name on dashboard */
  name: string;
  /* Display icon on dashboard */
  icon?: URL | WalkxcodeIcon | TargetRelativeIconPath;
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

/**
 * @pattern ^(png|svg|webp)\\/[\\w\\d\\-_]+\\.\\1$
 */
export type WalkxcodeIcon = string;

/**
 * @pattern ^@target/.+$
 */
export type TargetRelativeIconPath = string;
