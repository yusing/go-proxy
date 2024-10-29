package health

import (
	"github.com/yusing/go-proxy/internal/logging"
)

var logger = logging.With().Str("module", "health_mon").Logger()
