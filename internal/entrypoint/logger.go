package entrypoint

import (
	"github.com/yusing/go-proxy/internal/logging"
)

var logger = logging.With().Str("module", "entrypoint").Logger()
