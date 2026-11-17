package exposure

import (
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const kongBaseFixture = `_format_version: "3.0"
_transform: true
services:
  - name: admin-api
    url: http://obelisk-controller:8080
    routes:
      - name: admin-api
        paths: ["/api"]
        strip_path: false
  - name: rustfs-s3
    url: http://obelisk-rustfs:9000
    routes:
      - name: rustfs-s3
        paths: ["/s3"]
`

func TestMergeRustFSRoutes_DisabledRemovesRoute(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "default", Type: "rustfs", Enabled: false,
	}}
	out, err := mergeRustFSRoutes([]byte(kongBaseFixture), spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "rustfs-s3") {
		t.Fatalf("expected rustfs route removed, got:\n%s", s)
	}
}

func TestMergeRustFSRoutes_EnabledAddsRoute(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "default", Type: "rustfs", Enabled: true,
	}}
	base := `_format_version: "3.0"
_transform: true
services:
  - name: admin-api
    url: http://obelisk-controller:8080
    routes:
      - name: admin-api
        paths: ["/api"]
`
	out, err := mergeRustFSRoutes([]byte(base), spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "url: http://obelisk-rustfs:9000") {
		t.Fatalf("missing rustfs upstream:\n%s", s)
	}
	if !strings.Contains(s, "/s3") {
		t.Fatalf("missing /s3 path:\n%s", s)
	}
}

func TestKongYAMLSemanticallyEqual_IgnoresFormatting(t *testing.T) {
	a := `services:
  - name: admin-api
    url: http://obelisk-controller:8080
    routes:
      - name: admin-api
        paths: ["/api"]
`
	b := "services:\n  - name: admin-api\n    url: http://obelisk-controller:8080\n    routes:\n      - {name: admin-api, paths: [/api]}\n"
	if !kongYAMLSemanticallyEqual([]byte(a), []byte(b)) {
		t.Fatal("expected semantic equality")
	}
}

func TestMergeRustFSRoutes_NonDefaultInstancePath(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "iarchive01", Type: "rustfs", Enabled: true,
	}}
	out, err := mergeRustFSRoutes([]byte(`_format_version: "3.0"
_transform: true
services: []
`), spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "obelisk-rustfs-iarchive01:9000") {
		t.Fatalf("missing release-specific upstream:\n%s", s)
	}
	if !strings.Contains(s, "/s3/iarchive01") {
		t.Fatalf("missing instance path:\n%s", s)
	}
}

func TestMergeRustFSRoutes_AutoIDPath(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "ia1b2c3d4e5f", Type: "rustfs", Enabled: true,
	}}
	out, err := mergeRustFSRoutes([]byte(`_format_version: "3.0"
_transform: true
services: []
`), spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "/s3/ia1b2c3d4e5f") {
		t.Fatalf("expected id in path:\n%s", s)
	}
	if !strings.Contains(s, "route-rustfs-ia1b2c3d4e5f") {
		t.Fatalf("expected route service name:\n%s", s)
	}
}