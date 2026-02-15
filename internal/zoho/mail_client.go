package zoho

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"

	"github.com/SeMmyT/zohcli/internal/config"
)

// MailClient wraps the Zoho Client with mail-specific functionality
type MailClient struct {
	client    *Client
	accountID string // Cached primary account ID
}

// NewMailClient creates a new MailClient with the given config and token source
// It automatically resolves and caches the primary account ID
func NewMailClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*MailClient, error) {
	client, err := NewClient(cfg, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	mc := &MailClient{
		client: client,
	}

	// Resolve primary account ID
	ctx := context.Background()
	accountID, err := mc.getPrimaryAccountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get primary account ID: %w", err)
	}
	mc.accountID = accountID

	return mc, nil
}

// getPrimaryAccountID fetches the primary account ID from the Zoho Mail API
func (mc *MailClient) getPrimaryAccountID(ctx context.Context) (string, error) {
	resp, err := mc.client.DoMail(ctx, http.MethodGet, "/api/accounts", nil)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", mc.parseErrorResponse(resp)
	}

	var accountResp MailAccountListResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if accountResp.Status.Code != 200 {
		return "", fmt.Errorf("API error: %s (code %d)", accountResp.Status.Description, accountResp.Status.Code)
	}

	if len(accountResp.Data) == 0 {
		return "", fmt.Errorf("no mail accounts found")
	}

	return accountResp.Data[0].AccountID, nil
}

// parseErrorResponse attempts to parse an error response from the Zoho Mail API
func (mc *MailClient) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return the raw body
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// If we successfully parsed an APIError, use its Error() method
	return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
}

// ListFolders fetches all folders for the primary account
func (mc *MailClient) ListFolders(ctx context.Context) ([]Folder, error) {
	path := fmt.Sprintf("/api/accounts/%s/folders", mc.accountID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var folderResp FolderListResponse
	if err := json.NewDecoder(resp.Body).Decode(&folderResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if folderResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", folderResp.Status.Description, folderResp.Status.Code)
	}

	return folderResp.Data, nil
}

// GetFolderByName finds a folder by name (case-insensitive)
func (mc *MailClient) GetFolderByName(ctx context.Context, name string) (*Folder, error) {
	folders, err := mc.ListFolders(ctx)
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		if strings.EqualFold(folder.FolderName, name) {
			return &folder, nil
		}
	}

	return nil, fmt.Errorf("folder not found: %s", name)
}

// ListLabels fetches all labels for the primary account
func (mc *MailClient) ListLabels(ctx context.Context) ([]Label, error) {
	path := fmt.Sprintf("/api/accounts/%s/labels", mc.accountID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var labelResp LabelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&labelResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if labelResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", labelResp.Status.Description, labelResp.Status.Code)
	}

	return labelResp.Data, nil
}

// ListMessages fetches messages from a folder with pagination
func (mc *MailClient) ListMessages(ctx context.Context, folderID string, start, limit int) ([]MessageSummary, error) {
	path := fmt.Sprintf("/api/accounts/%s/messages/view?folderId=%s&start=%d&limit=%d",
		mc.accountID, folderID, start, limit)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var messageResp MessageListResponse
	if err := json.NewDecoder(resp.Body).Decode(&messageResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if messageResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", messageResp.Status.Description, messageResp.Status.Code)
	}

	return messageResp.Data, nil
}

// GetMessageMetadata fetches full metadata for a specific message
func (mc *MailClient) GetMessageMetadata(ctx context.Context, folderID, messageID string) (*MessageMetadata, error) {
	path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/details",
		mc.accountID, folderID, messageID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var metadataResp MessageMetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&metadataResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if metadataResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", metadataResp.Status.Description, metadataResp.Status.Code)
	}

	return &metadataResp.Data, nil
}

// GetMessageContent fetches the HTML body content for a specific message
func (mc *MailClient) GetMessageContent(ctx context.Context, folderID, messageID string) (*MessageContent, error) {
	path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/content",
		mc.accountID, folderID, messageID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var contentResp MessageContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if contentResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", contentResp.Status.Description, contentResp.Status.Code)
	}

	return &contentResp.Data, nil
}

// SearchMessages searches messages using Zoho search syntax
func (mc *MailClient) SearchMessages(ctx context.Context, searchKey string, start, limit int) ([]MessageSummary, error) {
	path := fmt.Sprintf("/api/accounts/%s/messages/search?searchKey=%s&start=%d&limit=%d",
		mc.accountID, url.QueryEscape(searchKey), start, limit)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var messageResp MessageListResponse
	if err := json.NewDecoder(resp.Body).Decode(&messageResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if messageResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", messageResp.Status.Description, messageResp.Status.Code)
	}

	return messageResp.Data, nil
}

// GetThread fetches all messages in a thread by filtering folder messages
func (mc *MailClient) GetThread(ctx context.Context, folderID, threadID string, limit int) ([]MessageSummary, error) {
	if limit == 0 {
		limit = 200 // Default scan limit
	}

	var matchingMessages []MessageSummary
	scanned := 0
	start := 0
	pageSize := 200 // Max messages per API call

	for scanned < limit {
		// Calculate how many messages to fetch in this iteration
		remaining := limit - scanned
		fetchSize := pageSize
		if remaining < pageSize {
			fetchSize = remaining
		}

		messages, err := mc.ListMessages(ctx, folderID, start, fetchSize)
		if err != nil {
			return nil, err
		}

		// No more messages in folder
		if len(messages) == 0 {
			break
		}

		// Filter by thread ID
		for _, msg := range messages {
			if msg.ThreadID == threadID {
				matchingMessages = append(matchingMessages, msg)
			}
		}

		scanned += len(messages)
		start += len(messages)

		// If we got fewer messages than requested, we've reached the end
		if len(messages) < fetchSize {
			break
		}
	}

	if len(matchingMessages) == 0 {
		return nil, fmt.Errorf("no messages found for thread ID: %s", threadID)
	}

	return matchingMessages, nil
}

// ListAttachments fetches all attachments for a message
func (mc *MailClient) ListAttachments(ctx context.Context, folderID, messageID string) ([]Attachment, error) {
	path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/attachments",
		mc.accountID, folderID, messageID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var attachmentResp AttachmentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&attachmentResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if attachmentResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", attachmentResp.Status.Description, attachmentResp.Status.Code)
	}

	return attachmentResp.Data, nil
}

// DownloadAttachment downloads an attachment to a file
func (mc *MailClient) DownloadAttachment(ctx context.Context, folderID, messageID, attachmentID, destPath string) error {
	path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/attachments/%s",
		mc.accountID, folderID, messageID, attachmentID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mc.parseErrorResponse(resp)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", destPath, err)
	}

	// Stream response to file
	_, err = io.Copy(file, resp.Body)
	closeErr := file.Close()

	if err != nil {
		// Best-effort cleanup of partial download
		os.Remove(destPath)
		return fmt.Errorf("download failed: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("close file: %w", closeErr)
	}

	return nil
}
