package homepage

// PredefinedCategories by alias or docker image name.
var PredefinedCategories = map[string]string{
	"sonarr":       "Torrenting",
	"radarr":       "Torrenting",
	"bazarr":       "Torrenting",
	"lidarr":       "Torrenting",
	"readarr":      "Torrenting",
	"prowlarr":     "Torrenting",
	"watcharr":     "Torrenting",
	"qbittorrent":  "Torrenting",
	"qbit":         "Torrenting",
	"qbt":          "Torrenting",
	"transmission": "Torrenting",

	"jellyfin":   "Media",
	"jellyseerr": "Media",
	"emby":       "Media",
	"plex":       "Media",
	"navidrome":  "Media",
	"immich":     "Media",
	"tautulli":   "Media",
	"nextcloud":  "Media",
	"invidious":  "Media",

	"uptime":             "Monitoring",
	"uptime-kuma":        "Monitoring",
	"prometheus":         "Monitoring",
	"grafana":            "Monitoring",
	"netdata":            "Monitoring",
	"changedetection.io": "Monitoring",
	"changedetection":    "Monitoring",
	"influxdb":           "Monitoring",
	"influx":             "Monitoring",
	"dozzle":             "Monitoring",

	"adguardhome":  "Networking",
	"adgh":         "Networking",
	"adg":          "Networking",
	"pihole":       "Networking",
	"flaresolverr": "Networking",

	"homebridge":     "Home Automation",
	"home-assistant": "Home Automation",

	"dockge":       "Container Management",
	"portainer-ce": "Container Management",
	"portainer-be": "Container Management",

	"rss":        "RSS",
	"rsshub":     "RSS",
	"rss-bridge": "RSS",
	"miniflux":   "RSS",
	"freshrss":   "RSS",

	"paperless":     "Documents",
	"paperless-ngx": "Documents",
	"s-pdf":         "Documents",

	"minio":       "Storage",
	"filebrowser": "Storage",
	"rclone":      "Storage",
}
