import { IdleWatcherConfig } from "./providers/idlewatcher";
import { Route } from "./providers/routes";

//FIXME: fix this
export type DockerRoutes = {
  [key: string]: Route & IdleWatcherConfig;
};
