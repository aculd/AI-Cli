// message.go - Defines the Message struct for representing chat messages across the application.
// This struct is used for all chat message storage, transmission, and display.

package models

// Message represents a chat message.
type Message struct {
	Role          string `json:"role"`
	Content       string `json:"content"`
	MessageNumber int    `json:"message_number"`
}
