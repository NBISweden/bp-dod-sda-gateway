package dod_service_impl

import "github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"

func OriginDatasetFileLoader(v origin_dataset_file_loader.OriginDatasetFileLoader) func(*dodServiceImpl) {
	return func(impl *dodServiceImpl) {
		impl.originDatasetFileLoader = v
	}
}
