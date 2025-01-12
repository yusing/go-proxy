package autocert

import (
	"os"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func (p *Provider) Setup() (err E.Error) {
	if err = p.LoadCert(); err != nil {
		if !err.Is(os.ErrNotExist) { // ignore if cert doesn't exist
			return err
		}
		logger.Debug().Msg("obtaining cert due to error loading cert")
		if err = p.ObtainCert(); err != nil {
			return err
		}
	}

	for _, expiry := range p.GetExpiries() {
		logger.Info().Msg("certificate expire on " + strutils.FormatTime(expiry))
		break
	}

	return nil
}
