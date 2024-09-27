package autocert

import (
	"context"
	"os"

	E "github.com/yusing/go-proxy/error"
)

func (p *Provider) Setup(ctx context.Context) (err E.NestedError) {
	if err = p.LoadCert(); err != nil {
		if !err.Is(os.ErrNotExist) { // ignore if cert doesn't exist
			return err
		}
		logger.Debug("obtaining cert due to error loading cert")
		if err = p.ObtainCert(); err != nil {
			return err
		}
	}

	go p.ScheduleRenewal(ctx)

	for _, expiry := range p.GetExpiries() {
		logger.Infof("certificate expire on %s", expiry)
		break
	}

	return nil
}
