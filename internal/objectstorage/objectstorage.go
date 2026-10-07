package objectstorage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:generate mockgen -source=objectstorage.go -destination=../mocks/objectstorage_client_mock.go -package=mocks
type ObjectStorage interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}
