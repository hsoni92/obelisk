package adminapi

import (
	"net/http"
	"testing"
)

func TestIsPublic_DiscoverPath(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/discover/v1/projects/prj_abc", nil)
	if !isPublic(r) {
		t.Fatal("discovery GET should be public")
	}
	r2, _ := http.NewRequest(http.MethodPost, "/discover/v1/projects/prj_abc", nil)
	if isPublic(r2) {
		t.Fatal("discovery POST should not be public")
	}
}
