package common

var (
	WellKnownHTTPPorts = map[string]bool{
		"80":   true,
		"8000": true,
		"8008": true,
		"8080": true,
		"3000": true,
	}

	ServiceNamePortMapTCP = map[string]int{
		"mssql":            1433,
		"mysql":            3306,
		"mariadb":          3306,
		"postgres":         5432,
		"rabbitmq":         5672,
		"redis":            6379,
		"memcached":        11211,
		"mongo":            27017,
		"minecraft-server": 25565,

		"ssh":  22,
		"ftp":  21,
		"smtp": 25,
		"dns":  53,
		"pop3": 110,
		"imap": 143,
	}

	ImageNamePortMap = func() (m map[string]int) {
		m = make(map[string]int, len(ServiceNamePortMapTCP)+len(imageNamePortMap))
		for k, v := range ServiceNamePortMapTCP {
			m[k] = v
		}
		for k, v := range imageNamePortMap {
			m[k] = v
		}
		return
	}()

	imageNamePortMap = map[string]int{
		"adguardhome":         3000,
		"bazarr":              6767,
		"calibre-web":         8083,
		"changedetection.io":  3000,
		"dockge":              5001,
		"gitea":               3000,
		"gogs":                3000,
		"grafana":             3000,
		"home-assistant":      8123,
		"homebridge":          8581,
		"httpd":               80,
		"immich":              3001,
		"jellyfin":            8096,
		"lidarr":              8686,
		"microbin":            8080,
		"nginx":               80,
		"nginx-proxy-manager": 81,
		"open-webui":          8080,
		"plex":                32400,
		"portainer-be":        9443,
		"portainer-ce":        9443,
		"prometheus":          9090,
		"prowlarr":            9696,
		"radarr":              7878,
		"radarr-sma":          7878,
		"rsshub":              1200,
		"rss-bridge":          80,
		"sonarr":              8989,
		"sonarr-sma":          8989,
		"uptime-kuma":         3001,
		"whisparr":            6969,
	}
)
