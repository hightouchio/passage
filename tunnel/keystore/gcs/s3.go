package s3

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
)

// GCS is a Keystore for Google Cloud Storage
type GCS struct {
	Client     *storage.Client
	BucketName string
	KeyPrefix  string
}

func (k GCS) Get(ctx context.Context, id uuid.UUID) ([]byte, error) {
	object := k.Client.Bucket(k.BucketName).Object(k.KeyPrefix + id.String())
	reader, err := object.NewReader(ctx)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not get object reader")
	}
	defer reader.Close()

	keyData, err := io.ReadAll(reader)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not read contents")
	}

	return keyData, nil
}

func (k GCS) Set(ctx context.Context, id uuid.UUID, contents []byte) error {
	writer := k.Client.Bucket(k.BucketName).Object(k.KeyPrefix + id.String()).NewWriter(ctx)
	defer writer.Close()
	if _, err := writer.Write(contents); err != nil {
		return errors.Wrap(err, "could not write contents")
	}
	return nil
}

func (k GCS) Delete(ctx context.Context, id uuid.UUID) error {
	if err := k.Client.Bucket(k.BucketName).Object(k.KeyPrefix + id.String()).Delete(ctx); err != nil {
		return errors.Wrap(err, "could not delete object")
	}
	return nil
}
