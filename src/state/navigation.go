// state/navigation.go - NavigationStack using new ViewState protocol
// MIGRATION TARGET: legacy/gui.go (navigation stack logic)

package state

import (
	"aichat/src/types"
	"encoding/json"
	"sync"
)

// NavigationStack implements LIFO stack with main menu protection
// MIGRATION TARGET: internal/navigation/stack.go

type NavigationStack struct {
	stack []*types.ViewState
	mu    sync.RWMutex
}

// NewNavigationStack creates stack with main menu as root
func NewNavigationStack(mainMenu types.ViewState) *NavigationStack {
	if !mainMenu.IsMainMenu() {
		panic("root must be main menu")
	}
	root := mainMenu
	return &NavigationStack{stack: []*types.ViewState{&root}}
}

// Push adds state with special handling for main menu
func (ns *NavigationStack) Push(state types.ViewState) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if state.IsMainMenu() {
		root := state
		ns.stack = []*types.ViewState{&root}
		return
	}
	ns.stack = append(ns.stack, &state)
}

// Pop removes top state while protecting root
func (ns *NavigationStack) Pop() types.ViewState {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if len(ns.stack) <= 1 {
		return nil
	}
	top := ns.stack[len(ns.stack)-1]
	ns.stack = ns.stack[:len(ns.stack)-1]
	return *top
}

// Top returns current state without popping
func (ns *NavigationStack) Top() types.ViewState {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	if len(ns.stack) == 0 {
		return nil
	}
	return *ns.stack[len(ns.stack)-1]
}

// ReplaceTop replaces the top ViewState with a new one
func (ns *NavigationStack) ReplaceTop(state types.ViewState) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if len(ns.stack) > 0 {
		ns.stack[len(ns.stack)-1] = &state
	}
}

// SerializeStack converts entire stack to JSON
func (ns *NavigationStack) SerializeStack() ([]byte, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	var serialized []struct {
		Type  types.ViewType `json:"type"`
		State []byte         `json:"state"`
	}
	for _, s := range ns.stack {
		stateData, err := (*s).MarshalState()
		if err != nil {
			return nil, err
		}
		serialized = append(serialized, struct {
			Type  types.ViewType `json:"type"`
			State []byte         `json:"state"`
		}{
			Type:  (*s).ViewType(),
			State: stateData,
		})
	}
	return json.Marshal(serialized)
}

// DeserializeStack restores stack from JSON
func (ns *NavigationStack) DeserializeStack(data []byte) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	var serialized []struct {
		Type  types.ViewType `json:"type"`
		State []byte         `json:"state"`
	}
	if err := json.Unmarshal(data, &serialized); err != nil {
		return err
	}
	newStack := make([]*types.ViewState, 0, len(serialized))
	for _, item := range serialized {
		var state types.ViewState
		switch item.Type {
		case types.MenuStateType:
			state = &types.MenuViewState{}
		case types.ChatStateType:
			state = &types.ChatViewState{}
		case types.ModalStateType:
			state = &types.ModalViewState{}
		default:
			continue // Skip unknown types
		}
		if err := state.UnmarshalState(item.State); err != nil {
			return err
		}
		newStack = append(newStack, &state)
	}
	if len(newStack) > 0 && (*newStack[0]).IsMainMenu() {
		ns.stack = newStack
	}
	return nil
}
