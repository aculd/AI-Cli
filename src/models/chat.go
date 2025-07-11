package models

import "time"

// ChatMetadata stores additional information about a chat session.
type ChatMetadata struct {
	Summary    string    `json:"summary,omitempty"`
	Title      string    `json:"title,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	Model      string    `json:"model,omitempty"`
	Favorite   bool      `json:"favorite,omitempty"`
	ModifiedAt int64     `json:"modified_at,omitempty"` // Unix timestamp for last modification
}

// Chat represents a chat session for UI display
type Chat struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Favorite     bool         `json:"favorite"`
	UnreadCount  int          `json:"unread_count"`
	Metadata     ChatMetadata `json:"metadata"`
	LastMessage  string       `json:"last_message,omitempty"`
	LastModified int64        `json:"last_modified,omitempty"`
}

// ChatFile represents the complete chat file structure for JSON storage.
type ChatFile struct {
	Metadata ChatMetadata `json:"metadata"`
	Messages []Message    `json:"messages"`
}
