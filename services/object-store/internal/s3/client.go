package s3

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client *minio.Client
	mu     sync.RWMutex
}

type ObjectMetadata struct {
	Bucket      string
	Key         string
	Size        int64
	ContentType string
	LastModified time.Time
	VersionID   string
	ETag        string
}

type UploadProgress struct {
	UploadID string
	Part     int
	Size     int64
	ETag     string
}

func NewClient(ctx context.Context, endpoint string, accessKey string, secretKey string, useSSL bool) (*Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Client{
		client: minioClient,
	}, nil
}

func (c *Client) CreateBucketIfNotExists(ctx context.Context, bucket string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	exists, err := c.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
	}

	return nil
}

func (c *Client) PutObject(ctx context.Context, bucket string, key string, data io.Reader, size int64, contentType string) (*ObjectMetadata, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, err := c.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to put object %s in bucket %s: %w", key, bucket, err)
	}

	return &ObjectMetadata{
		Bucket:  bucket,
		Key:     key,
		Size:    info.Size,
		ETag:    info.ETag,
		VersionID: info.VersionID,
	}, nil
}

func (c *Client) GetObject(ctx context.Context, bucket string, key string) (io.ReadCloser, *ObjectMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	object, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object %s from bucket %s: %w", key, bucket, err)
	}

	stat, err := object.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat object: %w", err)
	}

	metadata := &ObjectMetadata{
		Bucket:      bucket,
		Key:         key,
		Size:        stat.Size,
		ContentType: stat.ContentType,
		LastModified: stat.LastModified,
		ETag:        stat.ETag,
	}

	return object, metadata, nil
}

func (c *Client) DeleteObject(ctx context.Context, bucket string, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", key, bucket, err)
	}

	return nil
}

func (c *Client) ListObjects(ctx context.Context, bucket string, prefix string) ([]ObjectMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var objects []ObjectMetadata

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range c.client.ListObjects(ctx, bucket, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		objects = append(objects, ObjectMetadata{
			Bucket:       bucket,
			Key:          object.Key,
			Size:         object.Size,
			ContentType:  object.ContentType,
			LastModified: object.LastModified,
		})
	}

	return objects, nil
}

func (c *Client) MultipartUploadInit(ctx context.Context, bucket string, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	upload, err := c.client.NewMultipartUpload(ctx, bucket, key, minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return upload.UploadID, nil
}

func (c *Client) MultipartUploadPart(ctx context.Context, bucket string, key string, uploadID string, partNumber int, data io.Reader, size int64) (*UploadProgress, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	part, err := c.client.PutObjectPart(ctx, bucket, key, uploadID, partNumber, data, size, minio.PutObjectPartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	return &UploadProgress{
		UploadID: uploadID,
		Part:     partNumber,
		Size:     size,
		ETag:     part.ETag,
	}, nil
}

func (c *Client) MultipartUploadComplete(ctx context.Context, bucket string, key string, uploadID string, parts []UploadProgress) (*ObjectMetadata, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	completeParts := make([]minio.CompletePart, len(parts))
	for i, part := range parts {
		completeParts[i] = minio.CompletePart{
			PartNumber: part.Part,
			ETag:       part.ETag,
		}
	}

	info, err := c.client.CompleteMultipartUpload(ctx, bucket, key, uploadID, completeParts, minio.CompleteMultipartUploadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return &ObjectMetadata{
		Bucket:    bucket,
		Key:       key,
		Size:      info.Size,
		ETag:      info.ETag,
		VersionID: info.VersionID,
	}, nil
}

func (c *Client) PresignedGetObject(ctx context.Context, bucket string, key string, expiration time.Duration) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	url, err := c.client.PresignedGetObject(ctx, bucket, key, expiration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

func (c *Client) PresignedPutObject(ctx context.Context, bucket string, key string, expiration time.Duration) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	url, err := c.client.PresignedPutObject(ctx, bucket, key, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return url.String(), nil
}

func (c *Client) SetObjectVersioning(ctx context.Context, bucket string, enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	config := minio.VersioningConfig{Status: "Suspended"}
	if enabled {
		config.Status = "Enabled"
	}

	return c.client.SetBucketVersioning(ctx, bucket, &config)
}

func (c *Client) SetObjectLifecycle(ctx context.Context, bucket string, retentionDays int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	config := minio.BucketLifecycle{
		Rules: []minio.LifecycleRule{
			{
				Filter: minio.Filter{
					Prefix: "",
				},
				Expiration: minio.Expiration{
					Days: retentionDays,
				},
				Status: "Enabled",
			},
		},
	}

	return c.client.SetBucketLifecycle(ctx, bucket, &config)
}

func (c *Client) Close() error {
	return nil
}
