package origin_dataset_sync

import (
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	metadataSubmitterURL string
	privateKey           []byte
	servicePort          int
	syncInterval         time.Duration
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "sync.metadata_submitter_sync_url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The url to the metadata submitter sync api")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				metadataSubmitterURL = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "sync.private_key",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "The private key used to sign token used when calling the metadata submitter sync API with ES256 algoritm")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				privateKey = []byte(viper.GetString(flagName))
			},
		},
		&config.Flag{
			Name: "sync.interval",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Duration(flagName, 1*time.Hour, "The interval to check for new origin datasets to be synced")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				syncInterval = viper.GetDuration(flagName)
			},
		},
		&config.Flag{
			Name: "sync.service_port",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 8081, "Port to host the grpc Origin Dataset Sync Service at")
			},
			AssignFunc: func(flagName string) {
				servicePort = viper.GetInt(flagName)
			},
		},
	)
}
