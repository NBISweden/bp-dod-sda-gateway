package dataset_on_demand_service

import (
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	servicePort      int
	userinfoEndpoint string
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "dod_service.port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 8080, "Port to host the grpc Dataset On Demand Service at")
			},
			AssignFunc: func(flagName string) {
				servicePort = viper.GetInt(flagName)
			},
		}, &config.Flag{
			Name: "dod_service.userinfo_endpoint",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Endpoint to hit to get the userinfo from the bearer token")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				userinfoEndpoint = viper.GetString(flagName)
			},
		},
	)
}
