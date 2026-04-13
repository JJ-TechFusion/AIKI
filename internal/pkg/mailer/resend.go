package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

type Sender interface {
	Send(ctx context.Context, request SendRequest) error
}

type SendRequest struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

type ResendSender struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResendSender(apiKey, fromEmail, fromName string) *ResendSender {
	from := strings.TrimSpace(fromEmail)
	if from != "" && strings.TrimSpace(fromName) != "" {
		from = fmt.Sprintf("%s <%s>", strings.TrimSpace(fromName), from)
	}

	return &ResendSender{
		apiKey: strings.TrimSpace(apiKey),
		from:   from,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *ResendSender) Send(ctx context.Context, request SendRequest) error {
	if s.apiKey == "" || s.from == "" {
		return fmt.Errorf("resend sender is not configured")
	}

	payload := map[string]any{
		"from":    s.from,
		"to":      request.To,
		"subject": request.Subject,
	}
	if strings.TrimSpace(request.HTML) != "" {
		payload["html"] = request.HTML
	}
	if strings.TrimSpace(request.Text) != "" {
		payload["text"] = request.Text
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		resendAPIURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	responseBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("resend send email failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
}
