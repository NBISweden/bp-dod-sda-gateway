package metadata_submitter

import (
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	metadataSubmitterURL string
	privateKey           []byte
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "metadata_submitter_sync_api.url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The url to the metadata submitter sync api")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				metadataSubmitterURL = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "metadata_submitter_sync_api.private_key",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The private key used to sign token used when calling the metadata submitter sync API with ES256 algoritm")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				privateKey = []byte(viper.GetString(flagName))
			},
		},
	)
}
