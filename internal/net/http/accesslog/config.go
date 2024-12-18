package accesslog

import "github.com/yusing/go-proxy/internal/utils"

type (
	Format  string
	Filters struct {
		StatusCodes LogFilter[*StatusCodeRange] `json:"status_codes"`
		Method      LogFilter[HTTPMethod]       `json:"method"`
		Headers     LogFilter[*HTTPHeader]      `json:"headers"` // header exists or header == value
		CIDR        LogFilter[*CIDR]            `json:"cidr"`
	}
	Fields struct {
		Headers FieldConfig `json:"headers"`
		Query   FieldConfig `json:"query"`
		Cookies FieldConfig `json:"cookies"`
	}
	Config struct {
		BufferSize uint    `json:"buffer_size" validate:"gte=1"`
		Format     Format  `json:"format" validate:"oneof=common combined json"`
		Path       string  `json:"path" validate:"required"`
		Filters    Filters `json:"filters"`
		Fields     Fields  `json:"fields"`
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
				Default: FieldModeDrop,
			},
			Query: FieldConfig{
				Default: FieldModeKeep,
			},
			Cookies: FieldConfig{
				Default: FieldModeDrop,
			},
		},
	}
}

func init() {
	utils.RegisterDefaultValueFactory(DefaultConfig)
}
