package objectstorage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:generate mockgen -source=s3.go -destination=../mocks/s3.go -package=mocks
type S3ObjectStorage interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}
