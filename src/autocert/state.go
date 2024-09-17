package autocert

type CertState int

const (
	CertStateValid    CertState = 0
	CertStateExpired  CertState = iota
	CertStateMismatch CertState = iota
)
