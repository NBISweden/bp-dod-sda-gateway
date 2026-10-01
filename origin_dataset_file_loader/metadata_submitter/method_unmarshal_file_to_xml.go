package metadata_submitter

import (
	"context"
	"errors"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
)

func (mdc *metadataSubmitterClient) UnmarshalFileToXml(ctx context.Context, file *origin_dataset_file_loader.FileInfo, dst any) error {
	entry, err := mdc.getDatasetMetadata(ctx, file.DatasetAccession)
	if err != nil {
		return err
	}

	switch file.MetadataFileType {
	case metadata_models.MetadataFileTypeDataset:
		*dst.(*metadata_models.DatasetSet) = *entry.originDataset.Dataset
	case metadata_models.MetadataFileTypeImage:
		*dst.(*metadata_models.ImageSet) = *entry.originDataset.Image
	case metadata_models.MetadataFileTypeAnnotation:
		*dst.(*metadata_models.AnnotationSet) = *entry.originDataset.Annotation
	case metadata_models.MetadataFileTypeObservation:
		*dst.(*metadata_models.ObservationSet) = *entry.originDataset.Observation
	case metadata_models.MetadataFileTypeObserver:
		*dst.(*metadata_models.ObserverSet) = *entry.originDataset.Observer
	case metadata_models.MetadataFileTypePolicy:
		*dst.(*metadata_models.PolicySet) = *entry.originDataset.Policy
	case metadata_models.MetadataFileTypeSample:
		*dst.(*metadata_models.SampleSet) = *entry.originDataset.Sample
	case metadata_models.MetadataFileTypeStaining:
		*dst.(*metadata_models.StainingSet) = *entry.originDataset.Staining
	default:
		return errors.New("unknown metadata file type")
	}

	return nil
}
