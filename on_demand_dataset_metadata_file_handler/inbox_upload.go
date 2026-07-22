package on_demand_dataset_metadata_file_handler

import (
	"context"
	"encoding/xml"
	"io"

	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/neicnordic/crypt4gh/streaming"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (dmfh *dodMetadataFileHandler) marshalEncryptAndUploadFile(ctx context.Context, filePath string, metadata any) error {
	ctx, span := observability.Tracer().Start(ctx, "marshalEncryptAndUploadFile", trace.WithAttributes(attribute.String("file-path", filePath)))
	defer span.End()

	reader, writer := io.Pipe()

	go func() {
		crypt4GHWriter, err := streaming.NewCrypt4GHWriterWithoutPrivateKey(writer, [][32]byte{dmfh.c4ghPublicKey}, nil)
		if err != nil {
			_ = writer.CloseWithError(err)

			return
		}

		enc := xml.NewEncoder(crypt4GHWriter)
		enc.Indent("", "  ")
		defer func() {
			_ = enc.Close()
		}()

		if err := enc.Encode(metadata); err != nil {
			_ = writer.CloseWithError(err)
			_ = crypt4GHWriter.Close()
		}
		_ = crypt4GHWriter.Close()

		_ = writer.Close()
	}()

	_, err := dmfh.transferManagerClient.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Body:   reader,
		Bucket: aws.String(dmfh.uploadUser),
		Key:    aws.String(filePath + ".c4gh"),
	})
	if err != nil {
		_ = reader.Close()

		return err
	}

	return reader.Close()
}
