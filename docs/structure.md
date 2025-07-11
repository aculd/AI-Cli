# Project Structure Deep Dive

## Directory & File Purpose and Relationships

### Top-Level
- **main.go**: The entry point for the application. Handles environment setup, onboarding (API key), and launches the Bubble Tea GUI. Delegates to the root app model for all further logic.
- **app.go**: Implements the root Bubble Tea model. Manages the navigation stack, delegates updates and views to the current state, and orchestrates global state transitions.
- **state.go**: (Stub) Placeholder for future centralized state management.
- **utils.go**: Shared utility functions for file operations, string manipulation, and other helpers used across the codebase.
- **errors.go**: Centralized error handling and logging utilities.
- **README.md**: User-facing documentation and quickstart guide.
- **.config/**: Stores user data and configuration in JSON files (e.g., chats, prompts, models, keys). Used by repositories for persistence.
- **.util/**: Dev/test resources, sample data, and mock configs for development and testing.

### components/
- **sidebar/**: (Planned/Partial) Handles sidebar navigation, including section headings, chat list, and favorites. Intended to manage focus, navigation, and display of chat-related metadata.
- **chat/**: Implements the main chat window, composite view, streaming logic, and tab management. Responsible for rendering chat messages, handling streaming AI responses, and managing chat-specific state.
- **input/**: Manages the input area at the bottom of the UI, including text input, command history, and editor logic. Handles user keystrokes and input focus.
- **modals/**: Contains modal dialog logic and the modal manager. Includes base modal logic, modal stack, and specific dialog types:
  - **dialogs/**: Confirmation, help, menu, list, editor, and dynamic notice modals.
  - **manager.go**: Manages the modal stack, ensuring correct push/pop and focus behavior.
  - **base.go**: Base modal logic, shared by all modal types.
- **menus/**: Contains menu logic (e.g., ChatMenu.go) for main and submenus. Handles menu state, entry selection, and menu-driven flows.
- **common/**: Shared UI utilities, such as centering, padding, and color helpers, used by multiple components.

### models/
- **chat.go**: Defines chat metadata and the chat file structure for persistence.
- **message.go**: Defines the Message struct for chat messages, used throughout the app.
- **prompt.go**: Defines prompt templates for the assistant.
- **model.go**: Defines AI model configurations and metadata.
- **key.go**: Defines API key structures and related metadata.

### services/
- **storage/**: Implements data persistence using the repository pattern. Each repository (chats, models, prompts, keys) provides CRUD operations and abstracts JSON file I/O. Includes JSON storage helpers and migration logic.
- **ai/**: Handles AI API integration, streaming, and provider abstraction. Responsible for sending/receiving messages to/from AI backends.
- **config/**: Manages configuration, encryption, and validation logic for secure and robust app operation.

### types/
- Contains shared enums, interfaces, and control info for menus, state, and navigation. Defines the ViewState interface, menu types, and control info mappings used throughout the app.

---

## How Components Interact

- **main.go** initializes the environment and launches the app by creating the root app model (from app.go) and starting the Bubble Tea event loop.
- **app.go** manages the navigation stack, delegating all updates and rendering to the current ViewState (which could be a menu, modal, chat view, etc.). It orchestrates transitions between menus, modals, and chat windows.
- **components/** modules (sidebar, chat, input, modals, menus, common) are responsible for rendering their respective UI areas and handling user input. They communicate state changes and navigation events via the navigation stack managed by app.go.
- **modals/** are pushed and popped from the modal stack, allowing for nested flows (e.g., confirmation, input, dynamic notice). The modal manager ensures only the topmost modal receives input.
- **models/** define the data structures used throughout the app. These are the source of truth for chat, message, prompt, model, and key data.
- **services/storage/** repositories are used by UI flows (e.g., chat creation, prompt selection) to persist and retrieve data from JSON files in .config/.
- **services/ai/** is called by chat components to send/receive messages from AI providers, supporting streaming and cancellation.
- **types/** provides the shared interfaces and enums that ensure all components and services can interact in a type-safe, consistent way.

---

## Patterns

### 1. **Stack-Based Navigation (State Stack)**
- All navigation (menus, modals, chat views) is managed via a stack. This enables robust back/forward navigation, predictable state restoration, and easy implementation of nested flows.

### 2. **Repository Pattern**
- All persistence (chats, models, prompts, keys) is handled via repositories, which abstract file I/O and provide CRUD operations. This decouples UI logic from storage details and enables future backend changes.

### 3. **Component Isolation**
- Each UI component (sidebar, chat, input, modal) is self-contained, with its own state, rendering, and input handling. Communication is via navigation stack events and shared interfaces, maximizing maintainability and testability.

### 4. **Modal Stack Pattern**
- Modals are managed via a stack, supporting nested and interruptible flows (e.g., confirmation, input, dynamic notice). Only the topmost modal receives input, and modals can be pushed/popped as needed.

### 5. **Service Layer Abstraction**
- All business logic (AI, storage, config) is encapsulated in service layers, keeping UI code clean and focused on presentation and interaction.

### 6. **Modern TUI/UX with Bubble Tea & Lipgloss**
- The UI is built using Bubble Tea for event-driven state management and Lipgloss for layout, color, and padding. This enables a modern, accessible, and visually appealing TUI.

### 7. **Error Handling via Modals**
- All errors and onboarding flows are handled via modals, ensuring consistent user experience and clear recovery paths.

---

This structure and these patterns ensure the project is modular, maintainable, extensible, and easy for new developers to understand and contribute to. 