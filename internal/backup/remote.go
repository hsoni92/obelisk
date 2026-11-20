package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var awsS3HostRegionRE = regexp.MustCompile(`^s3\.([a-z0-9-]+)\.amazonaws\.com$`)

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
	return strings.ToLower(u.Host), u.Scheme == "https", nil
}

// awsRegionFromEndpointHost returns the AWS region implied by an S3 endpoint host.
func awsRegionFromEndpointHost(host string) (region string, ok bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "s3.amazonaws.com" {
		return "us-east-1", true
	}
	m := awsS3HostRegionRE.FindStringSubmatch(host)
	if len(m) == 2 {
		return m[1], true
	}
	return "", false
}

func makeBucketOptions(host string) minio.MakeBucketOptions {
	if region, ok := awsRegionFromEndpointHost(host); ok {
		return minio.MakeBucketOptions{Region: region}
	}
	if strings.HasSuffix(host, ".r2.cloudflarestorage.com") {
		return minio.MakeBucketOptions{Region: "auto"}
	}
	return minio.MakeBucketOptions{}
}

func isBucketAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	switch resp.Code {
	case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
		return true
	default:
		return false
	}
}

func ensureRemoteBucket(ctx context.Context, cli *minio.Client, bucket, host string) (created bool, err error) {
	exists, err := cli.BucketExists(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("bucket head %q: %w", bucket, err)
	}
	if exists {
		return false, nil
	}
	opts := makeBucketOptions(host)
	if err := cli.MakeBucket(ctx, bucket, opts); err != nil {
		if isBucketAlreadyExists(err) {
			return false, nil
		}
		return false, fmt.Errorf(
			"create bucket %q (needs s3:CreateBucket or R2 bucket create): %w",
			bucket,
			err,
		)
	}
	exists, err = cli.BucketExists(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("verify bucket %q after create: %w", bucket, err)
	}
	if !exists {
		return false, fmt.Errorf("bucket %q still missing after create", bucket)
	}
	return true, nil
}

// TestRemote ensures the machine bucket exists (creating it when permitted),
// then verifies a preflight object upload. The second return is true when the
// bucket was created during this call.
func TestRemote(ctx context.Context, spec obelisk.Spec, creds RemoteCredentials) (bool, error) {
	dests := obelisk.EnabledBackupDestinations(spec)
	ep := strings.TrimSpace(spec.Backup.Remote.Endpoint)
	bucket := spec.BackupBucketName()
	if len(dests) > 0 {
		d := dests[0]
		ep = d.Endpoint
		bucket = obelisk.DestinationBucket(spec, d)
	}
	return TestRemoteEndpoint(ctx, ep, bucket, creds)
}

// TestRemoteEndpoint runs preflight against an arbitrary endpoint and bucket.
func TestRemoteEndpoint(ctx context.Context, endpoint, bucket string, creds RemoteCredentials) (bool, error) {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return false, err
	}
	if bucket == "" {
		return false, fmt.Errorf("bucket name is required")
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	created, err := ensureRemoteBucket(ctx, cli, bucket, host)
	if err != nil {
		return false, err
	}

	key := obelisk.BackupPreflightObject
	payload := strings.NewReader("ok")
	_, err = cli.PutObject(ctx, bucket, key, payload, int64(len("ok")), minio.PutObjectOptions{})
	if err != nil {
		return created, fmt.Errorf("test upload s3://%s/%s: %w", bucket, key, err)
	}
	_ = cli.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	return created, nil
}

// BucketUsage sums object sizes under Obelisk backup prefixes (best-effort, capped).
func BucketUsage(ctx context.Context, endpoint, bucket string, creds RemoteCredentials, maxObjects int) (bytes int64, objects int, err error) {
	if maxObjects <= 0 {
		maxObjects = 2000
	}
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return 0, 0, err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return 0, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	prefixes := []string{obelisk.BackupConfigPrefix, obelisk.BackupDataPrefix}
	for _, prefix := range prefixes {
		for obj := range cli.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}) {
			if obj.Err != nil {
				return bytes, objects, obj.Err
			}
			objects++
			bytes += obj.Size
			if objects >= maxObjects {
				return bytes, objects, nil
			}
		}
	}
	return bytes, objects, nil
}

// EnsureDestinationBucket creates the machine bucket when missing (same as Test connection).
func EnsureDestinationBucket(ctx context.Context, endpoint, bucket string, creds RemoteCredentials) error {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("bucket name is required")
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return err
	}
	_, err = ensureRemoteBucket(ctx, cli, bucket, host)
	return err
}

// PutObject uploads raw bytes to an object key.
func PutObject(ctx context.Context, endpoint, bucket, key string, creds RemoteCredentials, data []byte) error {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	_, err = cli.PutObject(ctx, bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentTypeForObjectKey(key),
	})
	if err != nil {
		return fmt.Errorf("upload s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

// DownloadPrefix copies objects under prefix into destDir, preserving key paths.
// Used by chained restore for the data/ prefix. config/ is not requested.
func DownloadPrefix(ctx context.Context, endpoint, bucket, prefix string, creds RemoteCredentials, destDir string) (int, error) {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return 0, err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	n := 0
	for obj := range cli.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return n, obj.Err
		}
		if obj.Key == "" || strings.HasSuffix(obj.Key, "/") {
			continue
		}
		if strings.HasPrefix(obj.Key, obelisk.BackupConfigPrefix) {
			continue
		}
		reader, err := cli.GetObject(ctx, bucket, obj.Key, minio.GetObjectOptions{})
		if err != nil {
			return n, err
		}
		dest := filepath.Join(destDir, filepath.FromSlash(obj.Key))
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			reader.Close()
			return n, err
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			reader.Close()
			return n, err
		}
		_, copyErr := io.Copy(f, reader)
		closeErr := f.Close()
		reader.Close()
		if copyErr != nil {
			return n, copyErr
		}
		if closeErr != nil {
			return n, closeErr
		}
		n++
	}
	return n, nil
}

// GetObject downloads an object body.
func GetObject(ctx context.Context, endpoint, bucket, key string, creds RemoteCredentials) ([]byte, error) {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	reader, err := cli.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get s3://%s/%s: %w", bucket, key, err)
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// DeleteObject removes one object key (idempotent).
func DeleteObject(ctx context.Context, endpoint, bucket, key string, creds RemoteCredentials) error {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return err
	}
	return cli.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// ListObjectKeys returns object keys under prefix up to maxKeys.
func ListObjectKeys(ctx context.Context, endpoint, bucket, prefix string, creds RemoteCredentials, maxKeys int) ([]string, error) {
	host, secure, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	cli, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(creds.AccessKeyID, creds.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	if maxKeys <= 0 {
		maxKeys = 5000
	}
	var keys []string
	for obj := range cli.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return keys, obj.Err
		}
		if obj.Key == "" || strings.HasSuffix(obj.Key, "/") {
			continue
		}
		keys = append(keys, obj.Key)
		if len(keys) >= maxKeys {
			break
		}
	}
	return keys, nil
}

func contentTypeForObjectKey(key string) string {
	switch {
	case strings.HasSuffix(key, ".sql"):
		return "application/sql"
	case strings.HasSuffix(key, ".archive"), strings.HasSuffix(key, ".tar"), strings.HasSuffix(key, ".snapshot"):
		return "application/octet-stream"
	case strings.HasSuffix(key, ".json"):
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
