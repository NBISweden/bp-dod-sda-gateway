package observability

import (
	"github.com/NBISweden/bp-dod-sda-gateway/internal/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	enabled bool
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "observability.enabled",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Bool(flagName, true, "If observability(metrics, tracing) is to be enabled")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				enabled = viper.GetBool(flagName)
			},
		},
	)
}
