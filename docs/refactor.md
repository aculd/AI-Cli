# Project Refactor Overview

## 1. Current Project Structure

**Main Application Files:**
- `main.go`: Entry point, initializes and launches the Bubble Tea TUI.
- `app.go`: Root BubbleTea model, main application orchestration, and state/flow stack management.
- `state.go`: (Stub) Placeholder for centralized state management.
- `client.go`, `chatwindow.go`, `modal.go`: Legacy or stub files, some logic pending migration.

**Component-Based Organization:**
- `components/sidebar/`: (Planned) Sidebar navigation logic and rendering.
- `components/chat/`: (Planned) Chat window display, message handling, tabs, streaming.
- `components/input/`: (Planned) Input area, text processing, history.
- `components/modals/`: (Planned) Modal dialogs, popup management, modal stack.
- `components/menus/`: Menu action logic (e.g., `ChatMenu.go` for chat menu flows).
- `components/common/`: (Planned) Shared UI components/utilities.

**Service Layer:**
- `services/storage/`: Data persistence, file management, repositories (chats, models, prompts, keys).
- `services/ai/`: AI API integration, streaming, provider abstraction.
- `services/config/`: Configuration management, validation, encryption.

**Models and Types:**
- `models/`: Data structures for chats, messages, prompts, models, API keys.
- `types/`: Shared types, enums, interfaces (menu, control info, state, etc.).

**Legacy/Reference:**
- `gui.go`: Canonical source for all UI, menu, and interaction logic. All refactored code must match its behavior.

---

## 2. Target Architecture (from Design Doc)

- **State Management:**
  - `state.go`, `navigation.go`, `actions.go`, `views.go`, `dispatcher.go`: Centralized state, navigation stack, action definitions, view state, dispatcher.
- **Component Structure:**
  - `components/sidebar/`, `components/chat/`, `components/input/`, `components/modals/`, `components/common/`.
- **Service Layer:**
  - `services/storage/`, `services/ai/`, `services/config/` (with full repo, migration, backup, and provider support).
- **Models/Types:**
  - `models/` (chat, message, prompt, model, key), `types/` (shared enums, interfaces).
- **Modal System:**
  - `components/modals/manager.go`, `base.go`, `stack_modal.go`, `dialogs/`, `notices/`.
- **Navigation Stack:**
  - FIFO stack for view/modal state, protected main menu anchor, pop/push/clear operations, breadcrumbs.
- **UI Layout:**
  - Three-pane layout: sidebar (25%), chat window (center), input (bottom), with dynamic resizing and focus management.
- **Communication:**
  - Message-based navigation, custom tea.Msg types, event aggregation.
- **Testing/Fixtures:**
  - `.util/` for test data, mock models, prompts, chat files.

---

## 3. Differences & Reasoning

### **Missing/Partial in Current Project:**
- No `navigation.go`, `actions.go`, `views.go`, or `dispatcher.go` yet (navigation logic is in `app.go`).
- `components/sidebar/`, `components/chat/`, `components/input/`, `components/modals/` are planned but not fully implemented.
- Modal stack and flow logic is in progress (see `app.go` and `components/menus/ChatMenu.go`).
- No dynamic resizing, padding, or advanced Lipgloss layout yet.
- No `.util/` test fixtures or repo caching.
- Some service layer features (migration, backup, provider abstraction) are stubs or missing.
- No CLI utility for menu-to-control mapping sanity.
- No full test suite for navigation stack, menu state, or chat window.
- No breadcrumbs or animated transitions.

### **Reasoning:**
- The refactor is staged: core state/flow stack and menu extraction come first, followed by modularization of sidebar, chat, input, and modal systems.
- All new code references `gui.go` for UI and behavior parity; nothing is considered complete until tested against it.
- Some features (e.g., advanced layout, repo caching, fixtures) are deferred until core navigation and menu flows are stable.
- Testing and CLI utilities will be added after main flows are migrated.

### **Planned Additions:**
- Implement `navigation.go`, `actions.go`, `views.go`, `dispatcher.go` for full state/flow separation.
- Complete modularization of all components and modal flows.
- Add dynamic resizing, padding, and advanced Lipgloss layout.
- Add `.util/` test fixtures and repo caching.
- Implement CLI utility for menu-to-control mapping.
- Write comprehensive tests for navigation stack, menu state, chat window.
- Add breadcrumbs and animated transitions.

---

## 4. Modularization & Migration Strategy

- **Component Extraction:** Each UI/menu/modal feature is extracted from `gui.go` into its own module, with state and control info mapped centrally.
- **State/Flow Stack:** All navigation and modal flows use a FIFO stack for robust, predictable transitions and back navigation.
- **Menu/Control Info Mapping:** Menu types and control hints are mapped in `types.go` and supporting files for consistency.
- **Testing & Parity:** Every feature is tested for parity with `gui.go` before being marked complete.
- **Documentation:** All migration and refactor tasks are tracked in `tasklist.md` and referenced in code comments.

---

## 5. TODOs (from Architecture & Tasklist)

- [ ] Implement `navigation.go`, `actions.go`, `views.go`, `dispatcher.go` for state/flow separation
- [ ] Complete modularization of `components/sidebar/`, `components/chat/`, `components/input/`, `components/modals/`, `components/common/`
- [ ] Implement modal stack and flow logic for all menu/modal actions
- [ ] Add dynamic resizing, padding, and advanced Lipgloss layout
- [ ] Add `.util/` test fixtures and repo caching
- [ ] Implement CLI utility for menu-to-control mapping
- [ ] Write comprehensive tests for navigation stack, menu state, chat window
- [ ] Add breadcrumbs and animated transitions
- [ ] Complete service layer features (migration, backup, provider abstraction)
- [ ] Move UI types to `models/ui.go` as planned
- [ ] Add inline TODOs in `gui.go` linking to new modules
- [ ] Add header to `gui.go` marking it as canonical reference
- [ ] Mark each section in `gui.go` as “migrated” or “pending”
- [ ] Archive `gui.go` only after full test parity is achieved

---

## 6. Recent Changes

### MenuViewState Rendering Fix (Latest)
- **Issue**: MenuViewState was rendering unnecessary chat components (status bar, sidebar, chat window, input) alongside the menu overlay, creating a cluttered view.
- **Solution**: Modified `CompositeChatViewState.View()` to check the entire modal stack instead of just the Menu field, and render only the active modal when any modal is present.
- **Changes Made**:
  - Updated `View()` method to check `len(c.MenuStack) > 0` instead of `c.Menu != nil`
  - Added proper handling for all modal types: `MenuViewState`, `DynamicNoticeModal`, `InputPromptModal`, and `dialogs.ListModal`
  - Updated `Update()` method to handle all modal types in the stack, not just MenuViewState
  - Each modal type now renders in full-screen mode without underlying chat components
- **Result**: Clean, focused modal rendering that only shows the active modal without background clutter.

## 7. Summary

The refactor is ongoing, with a clear migration path from a monolithic `gui.go` to a modular, maintainable Bubble Tea architecture. All new code is tested for parity with the original implementation, and the navigation/modal stack ensures robust, predictable state transitions. The project is on track to achieve the target architecture, with all progress and tasks tracked in `tasklist.md`.
