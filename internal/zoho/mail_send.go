package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// UploadAttachment uploads a file and returns an attachment reference for use in send requests
func (mc *MailClient) UploadAttachment(ctx context.Context, filePath string) (*AttachmentReference, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Extract filename
	fileName := filepath.Base(filePath)

	// Build URL manually (bypass doRequest which sets application/json)
	uploadURL := mc.client.region.MailBase + fmt.Sprintf("/api/accounts/%s/messages/attachments?fileName=%s",
		mc.accountID, url.QueryEscape(fileName))

	// Create request with file body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, file)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// CRITICAL: Set Content-Type to application/octet-stream (NOT application/json)
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute via HTTP client (bypassing DoMail to avoid automatic JSON content-type)
	resp, err := mc.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var uploadResp AttachmentUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if uploadResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", uploadResp.Status.Description, uploadResp.Status.Code)
	}

	return &uploadResp.Data, nil
}

// SendEmail sends a new email message
func (mc *MailClient) SendEmail(ctx context.Context, req *SendEmailRequest) error {
	path := fmt.Sprintf("/api/accounts/%s/messages", mc.accountID)
	return mc.sendEmailRequest(ctx, path, req)
}

// ReplyToEmail replies to a message
func (mc *MailClient) ReplyToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
	req.Action = "reply"
	path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
	return mc.sendEmailRequest(ctx, path, req)
}

// ReplyAllToEmail replies to all recipients of a message
func (mc *MailClient) ReplyAllToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
	req.Action = "replyall"
	path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
	return mc.sendEmailRequest(ctx, path, req)
}

// ForwardEmail forwards a message to new recipients
func (mc *MailClient) ForwardEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
	req.Action = "forward"
	path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
	return mc.sendEmailRequest(ctx, path, req)
}

// sendEmailRequest is a private helper for all send operations
func (mc *MailClient) sendEmailRequest(ctx context.Context, path string, req *SendEmailRequest) error {
	// Marshal request to JSON
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// POST via DoMail (sets Content-Type: application/json)
	resp, err := mc.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mc.parseErrorResponse(resp)
	}

	var sendResp SendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if sendResp.Status.Code != 200 {
		return fmt.Errorf("API error: %s (code %d)", sendResp.Status.Description, sendResp.Status.Code)
	}

	return nil
}
