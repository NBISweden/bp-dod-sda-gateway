package dod_metadata_file_handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/observability"
	"github.com/neicnordic/crypt4gh/streaming"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (dmfh *dodMetadataFileHandler) marshalEncryptAndUploadFile(ctx context.Context, filePath string, metadata any) error {
	ctx, span := observability.Tracer().Start(ctx, "marshalEncryptAndUploadFile", trace.WithAttributes(attribute.String("file-path", filePath)))
	defer span.End()

	encryptedContent := bytes.Buffer{}
	crypt4GHWriter, err := streaming.NewCrypt4GHWriterWithoutPrivateKey(&encryptedContent, [][32]byte{dmfh.c4ghPublicKey}, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = crypt4GHWriter.Close()
	}()

	xmlContent, err := xml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}
	if _, err := crypt4GHWriter.Write(xmlContent); err != nil {
		return err
	}

	// TODO better c4gh writing without needing to read it all into a new reader
	_, err = dmfh.transferManagerClient.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Body:   bytes.NewReader(encryptedContent.Bytes()),
		Bucket: aws.String(dmfh.uploadUser),
		Key:    aws.String(filePath + ".c4gh"),
	})

	if err != nil {
		return err
	}

	return nil
}
