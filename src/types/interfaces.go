package types

// ViewType is an enum for different view state types.
type ViewType int

const (
	MenuStateType ViewType = iota
	ChatStateType
	ModalStateType
)

// ViewState interface is defined in view_state.go
// This file contains other interfaces and types
