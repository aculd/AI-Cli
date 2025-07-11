# Features Deep Dive

This document provides a detailed analysis of the feature set for each major view in the Go AI CLI/TUI project, along with the logic and interactions behind them. It is intended to help new developers understand the user-facing capabilities and the architectural reasoning for each part of the application.

---

## 1. Main Menu View

### **Features:**
- Central navigation hub for the application.
- Menu entries: Chats, Prompts, Models, API Keys, Help, Exit.
- All entries are centered, styled, and navigable with up/down/enter/esc.
- Disabled entries if prerequisites (e.g., API key) are missing.
- Control hints always visible (e.g., "Up/Down: Navigate, Enter: Select, Esc: Back").

### **Logic:**
- Managed as a stack-based view; always the anchor of the navigation stack.
- Selecting an entry pushes the corresponding submenu or flow onto the stack.
- Esc returns to the previous state or exits if at the anchor.
- Menu state is restored on return, preserving selection.

---

## 2. Chats Submenu View

### **Features:**
- Submenu for chat-related actions: New Chat, List Chats, List Favorites, Custom Chat, Delete Chat.
- Each entry launches a distinct flow:
  - **New Chat:** Prompts for a name, checks for duplicates, uses timestamp if blank, error notice if duplicate.
  - **List Chats/Favorites/Delete Chat:** Scrollable list modal (10 at a time), up/down navigation, enter to select, f to favorite/unfavorite, r to rename, esc to return.
  - **Custom Chat:** Guides user through selecting model, prompt, and API key, with animated dynamic notice for API key testing.
  - **Delete Chat:** Prompts for confirmation before deletion.
- All modals and menus are centered and styled.
- Control hints and error messages are clear and consistent.

### **Logic:**
- Each submenu entry pushes a new modal or flow onto the stack.
- Flows are composed of reusable modals (input, list, confirmation, dynamic notice).
- State is preserved between steps; cancellation returns to the submenu.
- Uses repository pattern for chat persistence.

---

## 3. Chat View (Composite View)

### **Features:**
- Main chat window with streaming AI responses.
- Each open chat has its own worker/goroutine for streaming.
- Supports cancellation (Ctrl+S), which removes the last assistant/user message and pre-fills the input box.
- Notification/badge support for background chat updates.
- Colorized, well-padded tags for user and assistant.
- Emoji for active chat, section headings, and response received.
- Sidebar integration for chat navigation.
- Input area for sending messages, with history and editing.

### **Logic:**
- Composite view manages sidebar, chat window, and input as subviews.
- Streaming is handled asynchronously; cancellation is robust.
- State transitions (e.g., closing chat) are managed via the navigation stack.
- All chat data is persisted via the chat repository.

---

## 4. Sidebar View

### **Features:**
- Displays active chats, all chats, and favorites with clear section headings and emoji.
- Up/down navigation, enter to select, esc/Ctrl+I to return focus to input.
- Focus can be cycled between sidebar and input using Tab, Ctrl+T, and Ctrl+N.
- Robust navigation and focus management.

### **Logic:**
- Sidebar state is managed independently but interacts with the main navigation stack.
- Selecting a chat in the sidebar pushes the corresponding chat view onto the stack.
- Focus management ensures smooth transitions between sidebar and input.

---

## 5. Input View

### **Features:**
- Text input area at the bottom of the UI.
- Supports single-line and multi-line input, with wrapping.
- Command history navigation (up/down).
- Control hints for available actions.
- Pre-filling of input box on cancellation or error.

### **Logic:**
- Input state is managed independently but can be updated by other views (e.g., chat cancellation).
- History is persisted and can be navigated with up/down keys.
- Input is validated before sending to the chat view.

---

## 6. Modal Views

### **Features:**
- **InputPromptModal:** Reusable, styled modal for text input, with instruction and control text, error messages, and single/multi-line support.
- **ListModal:** Scrollable, windowed list modal for option selection (10 at a time), with instruction/control text and robust navigation.
- **ConfirmationModal:** For confirmation dialogs (1-3 options), with clear messaging and navigation.
- **DynamicNoticeModal:** Animated feedback (e.g., "Testing..."), cycles through notices at a set interval, displays result (success/failure with emoji), and handles user confirmation.
- **EditorModal:** Read-only editor for previewing content (e.g., chat preview).

### **Logic:**
- All modals are managed via a stack; only the topmost modal receives input.
- Modals can be nested for complex flows (e.g., input → dynamic notice → confirmation).
- Each modal type implements the ViewState interface for consistent integration.
- Control hints and error handling are consistent across all modals.

---

## 7. Prompts, Models, and API Key Management Views

### **Features:**
- List, add, remove, and set default for prompts, models, and API keys.
- Scrollable list modals for selection.
- Input modals for adding new entries.
- Confirmation modals for deletion.
- Control hints and error messages throughout.

### **Logic:**
- Each management view uses the repository pattern for persistence.
- Flows are composed of reusable modals for input, selection, and confirmation.
- State is preserved between steps; cancellation returns to the previous menu.

---

## 8. Error Handling & Onboarding Views

### **Features:**
- Onboarding flow for first-time setup or missing API key.
- Information modals for guidance and error messages.
- Error modals for invalid input or failed operations.
- Dynamic notice modals for animated feedback during testing.

### **Logic:**
- Onboarding is triggered automatically if prerequisites are missing.
- All errors are handled via modals, ensuring clear user feedback and recovery options.
- State is restored after error handling, preserving user progress.

---

## 9. State Flow & Interactions

- All views are managed via a stack-based navigation system.
- Each view is self-contained but interacts with others via navigation events and shared state.
- Modals can interrupt any flow and are always stack-based.
- Data persistence is handled via repositories, abstracting file I/O from UI logic.
- Streaming, cancellation, and error handling are robust and user-friendly.

---

This document should provide a clear, detailed understanding of the feature set and logic behind each view in the project, enabling new developers to quickly grasp the user experience and architectural reasoning. 