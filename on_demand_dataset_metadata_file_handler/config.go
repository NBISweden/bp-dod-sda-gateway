package on_demand_dataset_metadata_file_handler

import (
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	sdaAPIUrl              string
	remsUrl                string
	remsUser               string
	remsKey                string
	remsDemoWorkflowId     int
	remsDemoOrganisationId string
	remsCategoryId         int

	sdaAPIPollInterval time.Duration

	inboxEndpointUrl  string
	inboxToken        string
	inboxDisableHTTPS bool

	c4ghPublicKeyFilePath string
)

func init() {
	config.RegisterFlags(
		&config.Flag{
			Name: "rems.url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Endpoint URL to the rems")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				remsUrl = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "rems.user",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "User to send to rems on x-rems-user-id header")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				remsUser = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "rems.key",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Key to send to rems on x-rems-api-key header")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				remsKey = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "rems.demo.workflow_id",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, -1, "Rems Workflow id to use for demo purposes, overrides the origin dataset workflow id")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				remsDemoWorkflowId = viper.GetInt(flagName)
			},
		},
		&config.Flag{
			Name: "rems.demo.organisation_id",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Rems Organisation id to use for demo purposes, overrides the origin dataset organisation id")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				remsDemoOrganisationId = viper.GetString(flagName)
			},
		}, &config.Flag{
			Name: "rems.rems_category_id",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Int(flagName, 0, "The rems category id On Demand Datasets will be created under, if not configured or configured as 0, On Demand Datasets will not be created under any category")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				remsCategoryId = viper.GetInt(flagName)
			},
		}, &config.Flag{
			Name: "sda_api.url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Endpoint URL to the sda api")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				sdaAPIUrl = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "sda_api.poll_interval",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Duration(flagName, 5*time.Minute, "Interval to poll the sda api for metadata files")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				sdaAPIPollInterval = viper.GetDuration(flagName)
			},
		},
		&config.Flag{
			Name: "inbox.endpoint_url",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Endpoint URL to the s3")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				inboxEndpointUrl = viper.GetString(flagName)
			},
		},
		&config.Flag{
			Name: "inbox.disable_https",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.Bool(flagName, false, "If the endpoint_url does not use HTTPS scheme")
			},
			Required: false,
			AssignFunc: func(flagName string) {
				inboxDisableHTTPS = viper.GetBool(flagName)
			},
		},
		&config.Flag{
			Name: "inbox.token",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Token to use when authentication to S3(session token)")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				inboxToken = viper.GetString(flagName)
			},
		}, &config.Flag{
			Name: "inbox.c4gh_public_key_file_path",
			RegisterFunc: func(flagSet *pflag.FlagSet, flagName string) {
				flagSet.String(flagName, "", "Path to the c4gh public key to be used for encrypting files uploaded to the inbox")
			},
			Required: true,
			AssignFunc: func(flagName string) {
				c4ghPublicKeyFilePath = viper.GetString(flagName)
			},
		},
	)
}
