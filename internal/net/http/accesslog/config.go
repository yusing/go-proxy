package accesslog

import "github.com/yusing/go-proxy/internal/utils"

type (
	Format  string
	Filters struct {
		StatusCodes LogFilter[*StatusCodeRange]
		Method      LogFilter[HTTPMethod]
		Headers     LogFilter[*HTTPHeader] // header exists or header == value
		CIDR        LogFilter[*CIDR]
	}
	Fields struct {
		Headers FieldConfig
		Query   FieldConfig
		Cookies FieldConfig
	}
	Config struct {
		BufferSize uint
		Format     Format `validate:"oneof=common combined json"`
		Path       string `validate:"required"`
		Filters    Filters
		Fields     Fields
	}
)

var (
	FormatCommon   Format = "common"
	FormatCombined Format = "combined"
	FormatJSON     Format = "json"
)

const DefaultBufferSize = 100

func DefaultConfig() *Config {
	return &Config{
		BufferSize: DefaultBufferSize,
		Format:     FormatCombined,
		Fields: Fields{
			Headers: FieldConfig{
				DefaultMode: FieldModeDrop,
			},
			Query: FieldConfig{
				DefaultMode: FieldModeKeep,
			},
			Cookies: FieldConfig{
				DefaultMode: FieldModeDrop,
			},
		},
	}
}

func init() {
	utils.RegisterDefaultValueFactory(DefaultConfig)
}
