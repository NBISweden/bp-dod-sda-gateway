package config

import (
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	dodServicePort         int
	dodAdminServicePort    int
	dodServiceJwtPubKeyUrl string
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "dod_service.port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 8080, "Port to host the grpc Dataset On Demand Service at")
			},
			AssignFunc: func(flagName string) {
				dodServicePort = viper.GetInt(flagName)
			},
		}, &config.Flag{
			Name: "dod_service.jwt_pub_key_url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The url to the jwk endpoint serving the public key used to verify signature of incoming bearer token")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				dodServiceJwtPubKeyUrl = viper.GetString(flagName)
			},
		}, &config.Flag{
			Name: "dod_admin_service.port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 8081, "Port to host the grpc Dataset On Demand Admin Service at")
			},
			AssignFunc: func(flagName string) {
				dodAdminServicePort = viper.GetInt(flagName)
			},
		},
	)
}

func DodServicePort() int {
	return dodServicePort
}
func DodServiceJwtPubKeyUrl() string {
	return dodServiceJwtPubKeyUrl
}
func DodAdminServicePort() int {
	return dodAdminServicePort
}
