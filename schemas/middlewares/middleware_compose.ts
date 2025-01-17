import { MiddlewaresMap } from "./middlewares";

export type MiddlewareComposeConfigBase = {
  use: keyof MiddlewaresMap;
};

/**
 * @additionalProperties false
 */
export type MiddlewareComposeConfig = (MiddlewareComposeConfigBase &
  Partial<MiddlewaresMap[keyof MiddlewaresMap]>)[];
