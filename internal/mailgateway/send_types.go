package mailgateway

import "strings"

type sendRequest struct {
	To             any    `json:"to"`
	Subject        string `json:"subject"`
	Text           string `json:"text"`
	HTML           string `json:"html"`
	AccountID      string `json:"account_id"`
	Failover       bool   `json:"failover"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (r sendRequest) firstTo() string {
	switch v := r.To.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	case []string:
		for _, s := range v {
			if strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

type sendResponse struct {
	AccountID string `json:"account_id"`
	From      string `json:"from"`
	Accepted  bool   `json:"accepted"`
	Deduped   bool   `json:"deduped,omitempty"`
}