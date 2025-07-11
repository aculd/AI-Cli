// manager.go - Contains ModalManager for managing a stack of modals and restoring previous state on close.

package modals

import "aichat/src/types"

// ModalManager manages a stack of modals for the application.
type ModalManager struct {
	stack []types.ViewState
}

// Push adds a modal to the top of the stack.
func (mm *ModalManager) Push(modal types.ViewState) {
	mm.stack = append(mm.stack, modal)
}

// Pop removes and returns the top modal from the stack.
func (mm *ModalManager) Pop() types.ViewState {
	if len(mm.stack) == 0 {
		return nil
	}
	m := mm.stack[len(mm.stack)-1]
	mm.stack = mm.stack[:len(mm.stack)-1]
	return m
}

// Current returns the top modal on the stack without removing it.
func (mm *ModalManager) Current() types.ViewState {
	if len(mm.stack) == 0 {
		return nil
	}
	return mm.stack[len(mm.stack)-1]
}
