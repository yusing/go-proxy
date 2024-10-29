package docker

import (
	"github.com/yusing/go-proxy/internal/logging"
)

var logger = logging.With().Str("module", "docker").Logger()
