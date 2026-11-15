package backup

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
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