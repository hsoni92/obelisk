package obelisk

import (
	"fmt"
	"strings"
)

// Stable member refs for exposure gateways in project instance_ids.
const (
	ExposureOpenRouterMember = "exposure/openrouter"
	ExposureMailMember       = "exposure/mail"
)

// IsExposureMemberRef reports whether ref is a built-in exposure gateway member.
func IsExposureMemberRef(ref string) bool {
	switch strings.TrimSpace(ref) {
	case ExposureOpenRouterMember, ExposureMailMember:
		return true
	default:
		return false
	}
}

// ExposureMemberLabel is a human label for admin UI.
func ExposureMemberLabel(ref string) string {
	switch ref {
	case ExposureOpenRouterMember:
		return "OpenRouter gateway"
	case ExposureMailMember:
		return "Mail gateway"
	default:
		return ref
	}
}

// ProjectDiscoveryExtras supplies gateway tokens for discovery (not in git-tracked spec).
type ProjectDiscoveryExtras struct {
	OpenRouterGatewayToken string
	MailGatewayToken       string
}

func exposureGatewayURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://%s:%s%s", PublicHost(), AdminHTTPPortString(), path)
}

func buildExposureConnection(ref string, extras ProjectDiscoveryExtras) (InstanceConnection, bool) {
	switch ref {
	case ExposureOpenRouterMember:
		url := exposureGatewayURL("/openrouter/api/v1")
		if url == "" {
			return InstanceConnection{}, false
		}
		ic := InstanceConnection{
			ID:          "gateway",
			Type:        "openrouter",
			DisplayName: ExposureMemberLabel(ref),
			RouteID:     ref,
			PublicPath:  url,
		}
		tok := strings.TrimSpace(extras.OpenRouterGatewayToken)
		if tok == "" {
			ic.Use = fmt.Sprintf("# OpenRouter gateway — token not configured on host\n%s", url)
		} else {
			ic.Use = strings.Join([]string{
				"# OpenRouter gateway (Bearer token)",
				"OPENROUTER_GATEWAY_URL=" + url,
				"OPENROUTER_GATEWAY_TOKEN=" + tok,
				"Authorization: Bearer " + tok,
			}, "\n")
		}
		return ic, true
	case ExposureMailMember:
		url := exposureGatewayURL("/mail/api/v1")
		if url == "" {
			return InstanceConnection{}, false
		}
		ic := InstanceConnection{
			ID:          "gateway",
			Type:        "mail",
			DisplayName: ExposureMemberLabel(ref),
			RouteID:     ref,
			PublicPath:  url,
		}
		tok := strings.TrimSpace(extras.MailGatewayToken)
		if tok == "" {
			ic.Use = fmt.Sprintf("# Mail gateway — token not configured on host\n%s", url)
		} else {
			ic.Use = strings.Join([]string{
				"# Mail gateway (Bearer token)",
				"MAIL_GATEWAY_URL=" + url,
				"MAIL_GATEWAY_TOKEN=" + tok,
				"Authorization: Bearer " + tok,
			}, "\n")
		}
		return ic, true
	default:
		return InstanceConnection{}, false
	}
}