# Project Progress & Architecture Analysis

## Overview

This document provides a comprehensive analysis of the current state, structure, and design rationale of the Go AI CLI/TUI project. It synthesizes the intended architecture (from `refactoring.md` and `refactor.md`) with the actual implementation in `src/`, `main.go`, and `app.go`. The goal is to educate new developers about the project’s logic, structure, and design decisions, enabling them to quickly understand and contribute to the codebase.

---

## 1. Project Structure: Intended vs. Actual

### **Intended Structure (from refactoring.md/refactor.md)**

The project is designed for strict modularity, separation of concerns, and testability, with the following key layers:

- **cmd/**: Entry point(s) for the application.
- **internal/**: Core application logic, navigation, state, and UI components.
  - **app/**: Root Bubble Tea model, state management, layout.
  - **navigation/**: Stack-based navigation, dispatcher, breadcrumbs.
  - **components/**: UI components (sidebar, chat, input, modals, menus, notices).
  - **services/**: Business logic (storage, AI, config, migrations, providers).
  - **models/**: Data structures (chat, message, prompt, model, UI state).
  - **types/**: Shared enums, interfaces, and control info.
- **pkg/**: Reusable utilities (e.g., Lipgloss helpers).
- **test/**: Automated tests and fixtures.
- **.util/**: Dev/test resources (sample data, mock configs).
- **legacy/**: Canonical reference implementation (`gui.go`).
- **docs/**: Architecture and migration documentation.

**Key Principles:**
- Navigation-centric, stack-based state management.
- Component isolation: each UI element is self-contained.
- Service layer for all persistence and business logic.
- All communication via navigation stack and defined interfaces.
- Testability and migration from legacy code.

### **Current Structure (src/)**

- **src/**: Main application code (no `cmd/` or `internal/` split yet).
  - **main.go**: Entry point, environment setup, onboarding, launches Bubble Tea app.
  - **app.go**: Root Bubble Tea model, navigation stack, state orchestration.
  - **components/**: Modular UI components.
    - **sidebar/**: Sidebar navigation (planned/partial).
    - **chat/**: Chat window, composite view, streaming, tabs.
    - **input/**: Input area, history, editor.
    - **modals/**: Modal dialogs, manager, base, dialogs (confirmation, help, menu, list), editor.
    - **menus/**: Menu logic (e.g., ChatMenu.go).
    - **common/**: Shared UI utilities.
  - **models/**: Data structures (chat, message, prompt, model, key).
  - **services/**: Business logic.
    - **storage/**: Repositories for chats, models, prompts, keys; JSON storage; migrations.
    - **ai/**: AI API integration, streaming, providers.
    - **config/**: Config management, encryption, validation.
  - **types/**: Shared types, enums, interfaces.
  - **state.go**: (Stub) Placeholder for centralized state management.
  - **utils.go**: Shared utility functions.
  - **errors.go**: Error handling.
  - **modals.go**: Legacy modal logic (pending migration).
  - **chats.go**: Legacy chat logic (pending migration).
  - **README.md**: Project overview and controls.
  - **.config/**: User data/config storage (JSON files).
  - **.util/**: Dev/test resources.

**Notable Differences:**
- No `cmd/` or `internal/` split yet; all code is under `src/`.
- Some legacy files (`modals.go`, `chats.go`) remain, pending migration.
- Navigation stack and modularization are in progress but not fully extracted.

---

## 2. File/Folder Descriptions

### **Top-Level**
- **main.go**: Entry point. Handles environment setup, onboarding (API key), launches the Bubble Tea GUI, and manages the main event loop.
- **app.go**: Root Bubble Tea model. Manages the navigation stack, delegates updates/views to the current state, and orchestrates global state.
- **state.go**: Placeholder for future centralized state management.
- **utils.go**: Shared utility functions (file ops, string utils, etc.).
- **errors.go**: Centralized error handling and logging.
- **README.md**: User-facing documentation.

### **components/**
- **sidebar/**: Sidebar navigation (section headings, chat list, favorites). Planned for robust focus and navigation.
- **chat/**: Main chat window, composite view, streaming, tabs. Implements chat state, message streaming, and chat-specific logic.
- **input/**: Input area, history, editor. Handles user input, command history, and text editing.
- **modals/**: Modal dialogs and manager. Includes base modal logic, modal stack, and specific dialogs (confirmation, help, menu, list, editor, dynamic notice).
- **menus/**: Menu logic (e.g., ChatMenu.go) for main and submenus.
- **common/**: Shared UI utilities (e.g., centering, padding, color helpers).

### **models/**
- **chat.go**: Chat metadata and file structure.
- **message.go**: Message struct for chat messages.
- **prompt.go**: Prompt templates.
- **model.go**: AI model definitions.
- **key.go**: API key definitions.

### **services/**
- **storage/**: Data persistence (repositories for chats, models, prompts, keys), JSON storage, migrations.
- **ai/**: AI API integration, streaming, provider abstraction.
- **config/**: Configuration management, encryption, validation.

### **types/**
- Shared enums, interfaces, and control info for menus, state, and navigation.

### **.config/**
- Stores user data and configuration in JSON files (chats, prompts, models, keys).

### **.util/**
- Dev/test resources, sample data, and mock configs.

---

## 3. Design Choices & Features

### **Navigation & State Management**
- **Stack-Based Navigation:**
  - All menus, modals, and chat views are managed via a stack, enabling robust back/forward navigation and predictable state restoration.
  - The main menu is always the anchor; popping the stack to empty returns to the main menu.
- **Component Isolation:**
  - Each UI component (sidebar, chat, input, modal) is self-contained, with its own state and rendering logic.
  - Communication between components is via navigation stack events and shared interfaces.
- **Modal System:**
  - Modals are managed via a stack, supporting nested flows (e.g., confirmation, input, dynamic notice).
  - Reusable modal types: input prompt, confirmation, scrollable list, dynamic notice, editor.

### **Service Layer & Persistence**
- **Repository Pattern:**
  - All data persistence (chats, models, prompts, keys) is handled via repositories, using JSON storage in `.config/`.
  - Repositories provide CRUD operations and abstract file I/O from the UI logic.
- **AI Integration:**
  - AI chat is integrated via a service layer, supporting streaming responses and provider abstraction.

### **UI/UX & Layout**
- **Modern TUI Design:**
  - Uses Bubble Tea and Lipgloss for layout, color, and padding.
  - All menus, modals, and chat windows are centered and styled for clarity and accessibility.
- **Scrollable List Modals:**
  - Windowed, scrollable lists for selecting prompts, models, API keys, and chats.
  - Instruction and control text for user guidance.
- **Dynamic Notice Modals:**
  - Animated feedback (e.g., “Testing...”) with cycling notices and result display (success/failure with emoji).
- **Control Hints:**
  - All UI elements display clear control hints (e.g., “Esc to cancel and return to Chats menu”).

### **Error Handling & Onboarding**
- **Onboarding Flow:**
  - On first run or missing API key, prompts user for key, title, and URL, with clipboard fallback.
  - Validates and tests API key before proceeding.
- **Error Modals:**
  - Consistent error handling via modals, with clear messages and recovery options.

### **Extensibility & Testability**
- **Modular Design:**
  - Each feature is implemented as a module, with clear interfaces and minimal coupling.
- **Test Fixtures:**
  - `.util/` and `test/` directories planned for test data and automated tests.
- **Legacy Reference:**
  - `gui.go` is preserved as a canonical reference for behavior parity during migration.

---

## 4. Program State Flow

1. **Startup:**
   - Initializes environment, error logging, and config directories/files.
   - Checks for API key; if missing, enters onboarding flow.
2. **Onboarding:**
   - Prompts for API key title, URL, and value (with clipboard fallback).
   - Validates and saves key, then tests it with a dynamic notice modal.
   - On success, proceeds to main menu; on failure, loops back.
3. **Main Menu:**
   - User can navigate to Chats, Prompts, Models, API Keys, Help, or Exit.
   - All navigation is stack-based; Esc always returns to the previous state.
4. **Chats Submenu:**
   - Options: New Chat, List Chats, List Favorites, Custom Chat, Delete Chat.
   - Each flow uses modals for input, selection, and confirmation.
   - Custom Chat flow: prompts for name, then uses scrollable list modals for prompt, model, and API key selection, with dynamic notice for API key testing.
5. **Chat View:**
   - Each chat opens in its own composite view, with streaming AI responses, cancellation, and real-time updates.
   - Sidebar and input area are integrated for robust navigation and focus management.
6. **Modals & Error Handling:**
   - All modals are stack-based, with clear control hints and error messages.
   - Dynamic notice modals provide animated feedback and result display.
7. **Persistence:**
   - All user data (chats, prompts, models, keys) is persisted in `.config/` as JSON.
   - Repositories abstract file I/O from UI logic.

---

## 5. Rationale & Decisions

- **Stack-Based Navigation:** Ensures robust, predictable user flows and easy restoration of previous state.
- **Component Isolation:** Maximizes maintainability, testability, and extensibility.
- **Repository Pattern:** Cleanly separates persistence from UI logic, enabling future backend changes.
- **Modern TUI/UX:** Prioritizes accessibility, clarity, and user guidance.
- **Migration from Legacy:** All new code is tested for parity with `gui.go` before being marked complete.
- **Testability:** Planned test fixtures and modularization support comprehensive testing.

---

## 6. Current Progress & Next Steps

- **Navigation stack and modularization are in progress; some legacy files remain.**
- **Sidebar, input, and modal systems are partially implemented; chat and menu flows are robust.**
- **Service layer and repository pattern are established for all core data types.**
- **Dynamic notice, scrollable list, and input modals are implemented and integrated.**
- **Onboarding and error handling flows are robust and user-friendly.**
- **Next Steps:**
  - Complete migration of legacy files and finalize modularization.
  - Implement centralized state management (`state.go`).
  - Expand test coverage and add fixtures.
  - Polish UI layout and add advanced Lipgloss features.
  - Continue feature parity checks with `gui.go`.

---

This document should provide new developers with a clear understanding of the project’s structure, logic, and design rationale, enabling effective onboarding and contribution. 