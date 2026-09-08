// Package email provides a thin Resend client plus a no-op sender for dev.
// Set RESEND_API_KEY to enable real delivery; omit it and a Noop sender is used.
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Sender is the only interface the rest of the app sees.
type Sender interface {
	Send(to, subject, html string) error
}

// ── Resend ────────────────────────────────────────────────────────────────────

type resendClient struct {
	apiKey string
	from   string
}

// NewResend returns a Sender that delivers via the Resend API.
// from should be a verified sender address, e.g. "Draftr <noreply@yourdomain.com>".
func NewResend(apiKey, from string) Sender {
	return &resendClient{apiKey: apiKey, from: from}
}

func (c *resendClient) Send(to, subject, html string) error {
	body, _ := json.Marshal(map[string]any{
		"from":    c.from,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	})
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ── Noop ─────────────────────────────────────────────────────────────────────

type noopSender struct{}

// NewNoop returns a Sender that logs but does not deliver.
func NewNoop() Sender { return &noopSender{} }

func (n *noopSender) Send(to, subject, _ string) error {
	fmt.Printf("[email noop] to=%s subject=%q\n", to, subject)
	return nil
}
