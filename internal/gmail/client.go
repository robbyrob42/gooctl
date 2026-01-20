package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps the Gmail API service
type Client struct {
	service *gmail.Service
	userID  string
}

// Message represents an email message
type Message struct {
	ID        string
	ThreadID  string
	Subject   string
	From      string
	To        string
	Date      time.Time
	Snippet   string
	Body      string
	IsUnread  bool
	Labels    []string
}

// NewClient creates a new Gmail client
func NewClient(ctx context.Context, httpClient *http.Client) (*Client, error) {
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &Client{
		service: service,
		userID:  "me",
	}, nil
}

// Search searches for messages matching the query
func (c *Client) Search(ctx context.Context, query string, maxResults int64) ([]*Message, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	call := c.service.Users.Messages.List(c.userID).Q(query).MaxResults(maxResults)
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	var messages []*Message
	for _, msg := range resp.Messages {
		fullMsg, err := c.GetMessage(ctx, msg.Id)
		if err != nil {
			continue // Skip messages we can't fetch
		}
		messages = append(messages, fullMsg)
	}

	return messages, nil
}

// GetMessage retrieves a full message by ID
func (c *Client) GetMessage(ctx context.Context, id string) (*Message, error) {
	msg, err := c.service.Users.Messages.Get(c.userID, id).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return parseMessage(msg), nil
}

// GetThread retrieves all messages in a thread
func (c *Client) GetThread(ctx context.Context, threadID string) ([]*Message, error) {
	thread, err := c.service.Users.Threads.Get(c.userID, threadID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	var messages []*Message
	for _, msg := range thread.Messages {
		messages = append(messages, parseMessage(msg))
	}

	return messages, nil
}

// Send sends a new email
func (c *Client) Send(ctx context.Context, to, subject, body string) (*Message, error) {
	// Get user's email for From header
	profile, err := c.service.Users.GetProfile(c.userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	raw := createRawEmail(profile.EmailAddress, to, subject, body, "")
	msg := &gmail.Message{
		Raw: raw,
	}

	sent, err := c.service.Users.Messages.Send(c.userID, msg).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return c.GetMessage(ctx, sent.Id)
}

// Reply replies to a message in a thread
func (c *Client) Reply(ctx context.Context, threadID, messageID, body string) (*Message, error) {
	// Get the original message for headers
	original, err := c.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// Get user's email for From header
	profile, err := c.service.Users.GetProfile(c.userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Build reply subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	raw := createRawEmail(profile.EmailAddress, original.From, subject, body, messageID)
	msg := &gmail.Message{
		Raw:      raw,
		ThreadId: threadID,
	}

	sent, err := c.service.Users.Messages.Send(c.userID, msg).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}

	return c.GetMessage(ctx, sent.Id)
}

// MarkAsRead marks a message as read
func (c *Client) MarkAsRead(ctx context.Context, id string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, id, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Context(ctx).Do()
	return err
}

// MarkAsUnread marks a message as unread
func (c *Client) MarkAsUnread(ctx context.Context, id string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, id, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}).Context(ctx).Do()
	return err
}

// Label represents a Gmail label
type Label struct {
	ID   string
	Name string
	Type string
}

// ListLabels returns all labels in the mailbox
func (c *Client) ListLabels(ctx context.Context) ([]*Label, error) {
	resp, err := c.service.Users.Labels.List(c.userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	var labels []*Label
	for _, l := range resp.Labels {
		labels = append(labels, &Label{
			ID:   l.Id,
			Name: l.Name,
			Type: l.Type,
		})
	}

	return labels, nil
}

// parseMessage converts a Gmail API message to our Message type
func parseMessage(msg *gmail.Message) *Message {
	m := &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Labels:   msg.LabelIds,
	}

	// Check if unread
	for _, label := range msg.LabelIds {
		if label == "UNREAD" {
			m.IsUnread = true
			break
		}
	}

	// Parse headers
	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			switch strings.ToLower(header.Name) {
			case "subject":
				m.Subject = header.Value
			case "from":
				m.From = header.Value
			case "to":
				m.To = header.Value
			case "date":
				if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					m.Date = t
				} else if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", header.Value); err == nil {
					m.Date = t
				}
			}
		}

		// Get body
		m.Body = getMessageBody(msg.Payload)
	}

	// Fallback for date from internal date
	if m.Date.IsZero() && msg.InternalDate > 0 {
		m.Date = time.Unix(msg.InternalDate/1000, 0)
	}

	return m
}

// getMessageBody extracts the body from a message payload
func getMessageBody(payload *gmail.MessagePart) string {
	if payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(decoded)
		}
	}

	// Check parts recursively
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			if part.Body != nil && part.Body.Data != "" {
				decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(decoded)
				}
			}
		}
		// Recurse into multipart
		if strings.HasPrefix(part.MimeType, "multipart/") {
			body := getMessageBody(part)
			if body != "" {
				return body
			}
		}
	}

	return ""
}

// createRawEmail creates a base64-encoded raw email
func createRawEmail(from, to, subject, body, inReplyTo string) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	if inReplyTo != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyTo))
		msg.WriteString(fmt.Sprintf("References: %s\r\n", inReplyTo))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return base64.URLEncoding.EncodeToString([]byte(msg.String()))
}
