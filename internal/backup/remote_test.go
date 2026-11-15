package backup

import "testing"

func TestAWSRegionFromEndpointHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		host   string
		region string
		ok     bool
	}{
		{"s3.amazonaws.com", "us-east-1", true},
		{"s3.eu-west-1.amazonaws.com", "eu-west-1", true},
		{"s3.US-EAST-2.amazonaws.com", "us-east-2", true},
		{"abc123.r2.cloudflarestorage.com", "", false},
		{"minio.example.com", "", false},
	}
	for _, tc := range cases {
		region, ok := awsRegionFromEndpointHost(tc.host)
		if ok != tc.ok || region != tc.region {
			t.Errorf("host %q: got (%q, %v), want (%q, %v)", tc.host, region, ok, tc.region, tc.ok)
		}
	}
}

func TestMakeBucketOptions(t *testing.T) {
	t.Parallel()
	aws := makeBucketOptions("s3.ap-south-1.amazonaws.com")
	if aws.Region != "ap-south-1" {
		t.Fatalf("aws region: got %q", aws.Region)
	}
	r2 := makeBucketOptions("a1b2.r2.cloudflarestorage.com")
	if r2.Region != "auto" {
		t.Fatalf("r2 region: got %q", r2.Region)
	}
	other := makeBucketOptions("s3.wasabisys.com")
	if other.Region != "" {
		t.Fatalf("generic region: got %q", other.Region)
	}
}