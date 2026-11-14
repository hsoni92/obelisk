package backup

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func parseEndpoint(raw string) (host string, secure bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("endpoint is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, err
	}
	if u.Host == "" {
		return "", false, fmt.Errorf("invalid endpoint %q", raw)
	}
	return u.Host, u.Scheme == "https", nil
}

// TestRemote verifies the machine bucket exists and accepts a preflight object.
func TestRemote(ctx context.Context, spec obelisk.Spec, creds RemoteCredentials) error {
	host, secure, err := parseEndpoint(spec.Backup.Remote.Endpoint)
	if err != nil {
		return err
	}
	bucket := spec.BackupBucketName()
	if bucket == "" {
		return fmt.Errorf("machine identity missing; reload config")
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	exists, err := cli.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("bucket head %q: %w", bucket, err)
	}
	if !exists {
		return fmt.Errorf(
			"create bucket %q in R2/S3 (name matches machine identity), then test again",
			bucket,
		)
	}
	key := obelisk.BackupPreflightObject
	payload := strings.NewReader("ok")
	_, err = cli.PutObject(ctx, bucket, key, payload, int64(len("ok")), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("test upload s3://%s/%s: %w", bucket, key, err)
	}
	_ = cli.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	return nil
}