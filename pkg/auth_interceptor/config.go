package auth_interceptor

import (
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	jwtPubKeyUrl      string
	jwtKeySetCacheTTL time.Duration
	allowedAudiences  []string
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "auth.jwt_pub_key_url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The url to the jwk endpoint serving the public key used to verify signature of incoming bearer token")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				jwtPubKeyUrl = viper.GetString(flagName)
			},
		}, &config.Flag{
			Name: "auth.jwt_key_set_cache_ttl",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Duration(flagName, time.Minute*60, "How long to cache the pub key fetched from the auth.jwt_pub_key_url")
			},
			AssignFunc: func(flagName string) {
				jwtKeySetCacheTTL = viper.GetDuration(flagName)
			},
		}, &config.Flag{
			Name: "auth.allowed_audiences",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.StringSlice(flagName, nil, "Comma-separated list of accepted JWT audiences (aud claim); a token must contain at least one of them")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				allowedAudiences = viper.GetStringSlice(flagName)
			},
		},
	)
}
