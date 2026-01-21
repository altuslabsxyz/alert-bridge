package dto

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestNewEphemeralResponse(t *testing.T) {
	text := "This is an ephemeral message"
	resp := NewEphemeralResponse(text)

	if resp.ResponseType != "ephemeral" {
		t.Errorf("ResponseType = %q, want %q", resp.ResponseType, "ephemeral")
	}
	if resp.Text != text {
		t.Errorf("Text = %q, want %q", resp.Text, text)
	}
	if resp.Blocks != nil {
		t.Error("Blocks should be nil")
	}
	if resp.Attachments != nil {
		t.Error("Attachments should be nil")
	}
}

func TestNewInChannelResponse(t *testing.T) {
	text := "This is an in-channel message"
	resp := NewInChannelResponse(text)

	if resp.ResponseType != "in_channel" {
		t.Errorf("ResponseType = %q, want %q", resp.ResponseType, "in_channel")
	}
	if resp.Text != text {
		t.Errorf("Text = %q, want %q", resp.Text, text)
	}
	if resp.Blocks != nil {
		t.Error("Blocks should be nil")
	}
	if resp.Attachments != nil {
		t.Error("Attachments should be nil")
	}
}

func TestNewEphemeralWithBlocks(t *testing.T) {
	text := "Fallback text"
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "Hello *world*", false, false),
			nil,
			nil,
		),
	}

	resp := NewEphemeralWithBlocks(text, blocks)

	if resp.ResponseType != "ephemeral" {
		t.Errorf("ResponseType = %q, want %q", resp.ResponseType, "ephemeral")
	}
	if resp.Text != text {
		t.Errorf("Text = %q, want %q", resp.Text, text)
	}
	if len(resp.Blocks) != 1 {
		t.Errorf("Blocks length = %d, want %d", len(resp.Blocks), 1)
	}
}

func TestNewEphemeralWithBlocks_EmptyBlocks(t *testing.T) {
	resp := NewEphemeralWithBlocks("text", []slack.Block{})

	if resp.ResponseType != "ephemeral" {
		t.Errorf("ResponseType = %q, want %q", resp.ResponseType, "ephemeral")
	}
	if len(resp.Blocks) != 0 {
		t.Errorf("Blocks length = %d, want %d", len(resp.Blocks), 0)
	}
}

func TestNewEphemeralWithBlocks_NilBlocks(t *testing.T) {
	resp := NewEphemeralWithBlocks("text", nil)

	if resp.ResponseType != "ephemeral" {
		t.Errorf("ResponseType = %q, want %q", resp.ResponseType, "ephemeral")
	}
	if resp.Blocks != nil {
		t.Error("Blocks should be nil when passed nil")
	}
}
