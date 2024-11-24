package types

type (
	Host      string
	Subdomain = Alias
)

func ValidateHost[String ~string](s String) (Host, error) {
	return Host(s), nil
}
