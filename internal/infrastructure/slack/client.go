package slack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
)

// Client wraps the Slack API client with domain-specific operations.
// Implements the alert.Notifier interface.
type Client struct {
	api            *slack.Client
	channelID      string
	messageBuilder *MessageBuilder
}

// NewClient creates a new Slack client.
func NewClient(botToken, channelID string, silenceDurations []time.Duration) *Client {
	return &Client{
		api:            slack.New(botToken),
		channelID:      channelID,
		messageBuilder: NewMessageBuilder(silenceDurations),
	}
}

// Notify sends an alert to Slack.
// Returns the message ID in the format "channel:timestamp".
func (c *Client) Notify(ctx context.Context, alert *entity.Alert) (string, error) {
	blocks := c.messageBuilder.BuildAlertMessage(alert)

	options := []slack.MsgOption{
		slack.MsgOptionBlocks(blocks...),
	}

	channelID, timestamp, err := c.api.PostMessageContext(ctx, c.channelID, options...)
	if err != nil {
		return "", fmt.Errorf("posting slack message: %w", err)
	}

	// Return channel:timestamp as message ID
	return fmt.Sprintf("%s:%s", channelID, timestamp), nil
}

// UpdateMessage updates an existing Slack message.
func (c *Client) UpdateMessage(ctx context.Context, messageID string, alert *entity.Alert) error {
	channelID, timestamp, err := parseMessageID(messageID)
	if err != nil {
		return err
	}

	var blocks []slack.Block
	if alert.IsActive() {
		blocks = c.messageBuilder.BuildAlertMessage(alert)
	} else {
		// For acked/resolved alerts, build without action buttons
		blocks = c.messageBuilder.BuildAckedMessage(alert)
	}

	options := []slack.MsgOption{
		slack.MsgOptionBlocks(blocks...),
	}

	_, _, _, err = c.api.UpdateMessageContext(ctx, channelID, timestamp, options...)
	if err != nil {
		return fmt.Errorf("updating slack message: %w", err)
	}

	return nil
}

// Name returns the notifier identifier.
func (c *Client) Name() string {
	return "slack"
}

// PostThreadReply posts a reply in a thread.
func (c *Client) PostThreadReply(ctx context.Context, messageID, text string) error {
	channelID, timestamp, err := parseMessageID(messageID)
	if err != nil {
		return err
	}

	options := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(timestamp),
	}

	_, _, err = c.api.PostMessageContext(ctx, channelID, options...)
	if err != nil {
		return fmt.Errorf("posting thread reply: %w", err)
	}

	return nil
}

// GetUserInfo retrieves user information by ID.
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*slack.User, error) {
	user, err := c.api.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user info: %w", err)
	}
	return user, nil
}

// GetUserEmail retrieves a user's email by their ID.
func (c *Client) GetUserEmail(ctx context.Context, userID string) (string, error) {
	user, err := c.GetUserInfo(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Profile.Email, nil
}

// AddReaction adds an emoji reaction to a message.
func (c *Client) AddReaction(ctx context.Context, messageID, emoji string) error {
	channelID, timestamp, err := parseMessageID(messageID)
	if err != nil {
		return err
	}

	err = c.api.AddReactionContext(ctx, emoji, slack.ItemRef{
		Channel:   channelID,
		Timestamp: timestamp,
	})
	if err != nil {
		return fmt.Errorf("adding reaction: %w", err)
	}

	return nil
}

// parseMessageID parses a message ID in the format "channel:timestamp".
func parseMessageID(messageID string) (channelID, timestamp string, err error) {
	parts := strings.SplitN(messageID, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid message ID format: %s", messageID)
	}
	return parts[0], parts[1], nil
}
