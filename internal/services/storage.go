package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
	client *minio.Client
	bucket string
}

func NewStorageService(endpoint, accessKey, secretKey, bucket string) (*StorageService, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	// Create bucket if not exists
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
	}

	return &StorageService{client: client, bucket: bucket}, nil
}

func (s *StorageService) SaveFile(ctx context.Context, filename, contentType string, data []byte) (string, error) {
	key := fmt.Sprintf("videos/%s", filename)
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *StorageService) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, nil)
	return u.String(), err
}

func (s *StorageService) GetObjectURL(objectKey string) string {
	return fmt.Sprintf("http://minio:9000/%s/%s", s.bucket, objectKey)
}

func (s *StorageService) SaveFileWithReader(ctx context.Context, filename, contentType string, pr io.Reader) (string, error) {
	key := fmt.Sprintf("videos/%s", filename)
	_, err := s.client.PutObject(ctx, s.bucket, key, pr, -1, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return key, nil
}
