//go:build integration

package objectstorage

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/require"
)

type r2Config struct {
	AccessKey       string `env:"R2_ACCESS_KEY,required"`
	SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY,required"`
	Bucket          string `env:"R2_BUCKET,required"`
	PublicURL       string `env:"R2_PUBLIC_URL,required"`
	S3ApiEndpoint   string `env:"S3_API_ENDPOINT,required"`
}

func TestR2(t *testing.T) {
	var cfg r2Config
	require.NoError(t, env.Parse(&cfg))

	httpClient := &http.Client{Timeout: 10 * time.Second}

	s3Client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretAccessKey, ""),
		Region:       "auto",
		BaseEndpoint: aws.String(cfg.S3ApiEndpoint),
		UsePathStyle: true,
		HTTPClient:   httpClient,
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	t.Run("put object", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		var jpegBuf bytes.Buffer

		jpeg.Encode(&jpegBuf, img, nil)

		_, err := s3Client.PutObject(ctx,
			&s3.PutObjectInput{
				Bucket:       aws.String(cfg.Bucket),
				Key:          aws.String("key"),
				Body:         &jpegBuf,
				ContentType:  aws.String("type"),
				CacheControl: aws.String("public, max-age=3600"),
			},
		)

		require.NoError(t, err)
	})

	t.Run("delete object", func(t *testing.T) {
		_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String("key"),
		})

		require.NoError(t, err)
	})
}
