package dataset_on_demand_admin

import "github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"

func OriginDatasetFileLoader(v origin_dataset_file_loader.OriginDatasetFileLoader) func(*serviceImpl) {
	return func(impl *serviceImpl) {
		impl.originDatasetFileLoader = v
	}
}
