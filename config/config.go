package config

import (
	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	dodServicePort int
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "dod-service-port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 8080, "Port to host the grpc Dataset On Demand service at")
			},
			AssignFunc: func(flagName string) {
				dodServicePort = viper.GetInt(flagName)
			},
		},
	)
}

func DodServicePort() int {
	return dodServicePort
}
