package navigation

import "aichat/src/types"

// NavigationAction enumerates navigation stack operations.
type NavigationAction int

const (
	PushAction NavigationAction = iota
	PopAction
	ResetAction
)

// NavigationMsg is a Bubble Tea message for navigation events.
type NavigationMsg struct {
	Action NavigationAction
	Target types.ViewState // Only for PushAction
}
