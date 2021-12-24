package s3

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io/ioutil"
)

type S3 struct {
	S3         *s3.S3
	BucketName string
	KeyPrefix  string
}

func (k S3) Get(ctx context.Context, id uuid.UUID) ([]byte, error) {
	response, err := k.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(k.BucketName),
		Key:    aws.String(k.KeyPrefix + id.String()),
	})
	if err != nil {
		return []byte{}, err
	}

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not read contents")
	}

	return contents, nil
}

func (k S3) Set(ctx context.Context, id uuid.UUID, contents []byte) error {
	_, err := k.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(k.BucketName),
		Key:    aws.String(k.KeyPrefix + id.String()),
		Body:   bytes.NewReader(contents),
	})
	return err
}

func (k S3) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := k.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(k.BucketName),
		Key:    aws.String(k.KeyPrefix + id.String()),
	})
	return err
}
