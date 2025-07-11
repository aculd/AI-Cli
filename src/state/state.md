# State Management Module Plan

This document outlines the architecture and purpose of all files in the `/state/` directory, as proposed in the project refactor and our ongoing chat. The state module is responsible for centralized, robust management of navigation, menu, modal, and view state throughout the Bubble Tea application.

---

## High-Level Overview

- **Centralized State:** All navigation, menu, and modal flows are managed via a centralized state module, ensuring predictable transitions and robust back navigation.
- **Navigation Stack:** A FIFO stack tracks the current view/modal hierarchy, with main menu anchor protection.
- **Menu State:** Each menu or submenu is represented by a `MenuState` struct, tracking selection and navigation history.
- **Control Info:** Control hints (key bindings, help) are mapped to each menu for consistent UI feedback.
- **Actions & Dispatcher:** Navigation and state changes are triggered by defined actions and coordinated by a dispatcher.
- **View State:** Each view (menu, modal, chat, etc.) has a corresponding state definition for modular updates and rendering.

---

## NavigationStack Design

```go
// NavigationStack manages the stack of view states (menus, modals, etc.)
type NavigationStack struct {
    stack    []ViewState // FIFO stack of view states
    maxDepth int         // Optional: maximum stack depth for safety
    // Guarantees:
    // 1. Always ≥1 item (MainMenu anchor is never popped)
    // 2. Thread-safe operations (mutex or channel for concurrent access)
}
```

### Description & Guarantees
- **Purpose:** Manages the stack of active views (menus, modals, dialogs, etc.) for navigation and state restoration.
- **Stack Invariant:**
  - The stack always contains at least one item (the MainMenu anchor). No operation can pop below this root.
  - **Special Behavior:** When pushing a new `MainMenu` state, the current stack is discarded and a new stack is instantiated with `MainMenu` as its root (full reset).
- **Thread Safety:** All stack operations (`Push`, `Pop`, `Clear`, etc.) must be thread-safe to support concurrent updates (consider using a mutex or channel).
- **Max Depth:** Optionally, `maxDepth` can be enforced to prevent runaway stack growth (e.g., in recursive modal flows).

### Key Operations
- `Push(ViewState)`: Adds a new view state to the top of the stack. If the view is `MainMenu`, resets the stack to only this state.
- `Pop() ViewState`: Removes and returns the top view state, but never pops below the MainMenu anchor.
- `Clear()`: Resets the stack to only the MainMenu anchor.
- `ResetStack()`: Alias for `Clear()`.
- `Top() ViewState`: Returns the current (top) view state.
- **TODO:** `PopTo(target ViewType)`, breadcrumbs, stack observers.

### Integration
- Used by the app root and dispatcher to manage all view transitions and modal flows.
- Each component (sidebar, chat, modals) interacts with the navigation stack for stateful navigation.

---

## File-by-File Plan

### 1. `navigation.go`
- **Purpose:** Implements the FIFO navigation stack for view/modal state.
- **Key Types:** `NavigationStack` (manages stack of `*MenuState`), stack operations (`Push`, `Pop`, `Clear`, `ResetStack`, `Top`).
- **Interactions:** Used by app root and dispatcher to manage view transitions and modal flows.
- **Status:** Implemented (initial version).
- **TODO:** Add `PopTo`, breadcrumbs, stack observers, and MainMenu push-reset logic.

### 2. `menu_state.go`
- **Purpose:** Defines the `MenuState` struct, tracking current menu, selection, and previous state for navigation.
- **Key Types:** `MenuState` (menu type, selected index, previous pointer), helpers for selection and navigation.
- **Interactions:** Used by navigation stack and all menu components.
- **Status:** TODO (to be implemented).

### 3. `control_info.go`
- **Purpose:** Maps control info (key hints, help lines) to each menu type for consistent UI display.
- **Key Types:** `ControlInfoType`, `ControlInfo`, `ControlInfoMap`, `MenuMeta`, `MenuMetas`.
- **Interactions:** Referenced by menu rendering and help dialogs.
- **Status:** TODO (to be implemented; currently in `types.go`).

### 4. `actions.go`
- **Purpose:** Defines navigation and state change actions (push, pop, clear, select, etc.) as typed events or functions.
- **Key Types:** Action enums/types, action dispatch functions.
- **Interactions:** Used by dispatcher and components to trigger state changes.
- **Status:** TODO (to be implemented).

### 5. `views.go`
- **Purpose:** Defines view state for each UI component (menus, modals, chat, input, etc.).
- **Key Types:** View state structs for each component, view state enums.
- **Interactions:** Used by app root and dispatcher to determine which view to render.
- **Status:** TODO (to be implemented).

### 6. `dispatcher.go`
- **Purpose:** Central coordinator for navigation and state transitions. Receives actions/events and updates the navigation stack and view state accordingly.
- **Key Types:** Dispatcher struct, event loop, action handlers.
- **Interactions:** Used by app root to process all navigation and state changes.
- **Status:** TODO (to be implemented).

---

## Modularization & Flow Stack Strategy

- All state transitions (menus, modals, dialogs, notices) are managed via the navigation stack and dispatcher.
- Each component (sidebar, chat, input, modals) interacts with the state module for navigation and state updates.
- The state module is the single source of truth for current view, navigation history, and modal flows.
- Robust back navigation and flow chaining are achieved via stack operations and state restoration.

---

## TODOs

- [ ] Implement `menu_state.go` with `MenuState` struct and helpers
- [ ] Move control info types/maps from `types.go` to `control_info.go`
- [ ] Implement `actions.go` for navigation/state actions
- [ ] Implement `views.go` for view state definitions
- [ ] Implement `dispatcher.go` for centralized state/event coordination
- [ ] Add advanced stack features: `PopTo`, breadcrumbs, observers
- [ ] Add MainMenu push-reset logic to navigation stack
- [ ] Document all new types and functions inline 