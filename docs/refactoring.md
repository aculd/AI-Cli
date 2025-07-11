Based on our discussion and architectural requirements, here's a refined folder structure that implements best practices while addressing the project's specific needs:

### Refactored Folder Structure
```
bubbletea-ai/
├── cmd/
│   └── bubbletea-ai/
│       └── main.go          # Entry point (minimal logic)
├── internal/
│   ├── app/                 # Application core
│   │   ├── app.go           # Root BubbleTea model
│   │   ├── state.go         # Centralized state management
│   │   └── layout.go        # Terminal layout composition
│   ├── navigation/          # Navigation subsystem
│   │   ├── stack.go         # FIFO stack implementation
│   │   ├── actions.go       # Push/Pop/Clear operations
│   │   ├── dispatcher.go    # State change coordination
│   │   └── breadcrumbs.go   # Navigation path tracking
│   ├── components/          # UI components
│   │   ├── sidebar/         # Left navigation pane
│   │   │   ├── model.go
│   │   │   ├── view.go
│   │   │   └── menu_stack.go
│   │   ├── chat/            # Main chat display
│   │   │   ├── model.go
│   │   │   ├── view.go
│   │   │   ├── tabs.go
│   │   │   └── streaming.go
│   │   ├── input/           # Bottom input area
│   │   │   ├── model.go
│   │   │   ├── view.go
│   │   │   └── history.go
│   │   └── modals/          # Overlay components
│   │       ├── manager.go
│   │       ├── base.go
│   │       ├── stack_modal.go
│   │       ├── dialogs/     # Specific dialog types
│   │       └── notices/     # Temporary notifications
│   ├── services/            # Business logic
│   │   ├── storage/
│   │   │   ├── repository.go
│   │   │   ├── json_store.go
│   │   │   ├── migrations/
│   │   │   └── backups/
│   │   ├── ai/
│   │   │   ├── client.go
│   │   │   ├── streaming.go
│   │   │   └── providers/   # OpenAI/Anthropic/etc.
│   │   └── config/
│   │       ├── manager.go
│   │       └── crypto.go
│   ├── models/              # Data structures
│   │   ├── chat.go
│   │   ├── message.go
│   │   ├── prompt.go
│   │   ├── model.go         # AI models
│   │   └── ui.go            # UI state types
│   └── types/               # Shared definitions
│       ├── enums.go         # MenuType, ControlInfoType
│       └── interfaces.go
├── pkg/                     # Reusable utilities
│   ├── lipgloss/
│   │   └── extensions.go    # Custom style helpers
│   └── tui/
│       └── responsive.go    # Layout utilities
├── test/                    # Automated tests
│   ├── navigation/
│   ├── components/
│   ├── services/
│   └── testdata/            # Fixtures
├── scripts/                 # Dev tools
│   └── control-mapper.go    # Menu-to-control CLI
├── .util/                   # Dev resources
│   ├── test_chats/          # Sample chat data
│   └── mock_models/         # Test configurations
├── legacy/                  # Reference implementation
│   └── gui.go               # Canonical behavior reference
└── docs/
    ├── refactor.md          # Architecture decisions
    └── tasklist.md          # Migration progress
```

### Key Improvements & Rationale

1. **Strict Separation of Concerns**
   - `internal/app`: Pure application orchestration
   - `internal/navigation`: Dedicated navigation subsystem
   - `internal/services`: Business logic without UI dependencies
   - `internal/components`: Pure presentation layer

2. **Navigation-Centric Architecture**
   ```go
   // internal/navigation/stack.go
   type Stack struct {
       views  []ViewState    // FIFO queue
       lock   sync.RWMutex   // Thread-safe access
   }

   func (s *Stack) Push(view ViewState) {
       s.lock.Lock()
       defer s.lock.Unlock()
       s.views = append(s.views, view)
   }

   func (s *Stack) Pop() (ViewState, error) {
       if len(s.views) <= 1 { // Prevent popping main menu
           return nil, ErrCannotPopMainMenu
       }
       // ... pop logic
   }
   ```

3. **Component Isolation**
   - Each component has:
     - Self-contained model
     - Dedicated view rendering
     - Isolated state management
   - Communication via navigation stack events only

4. **Service Layer Optimization**
   ```go
   // internal/services/storage/repository.go
   type ChatRepository interface {
       GetActive() (*models.Chat, error)
       Save(chat *models.Chat) error
       // ... other CRUD operations
   }
   ```

5. **Testability Enhancements**
   - Dedicated `test/` structure mirroring main code
   - `testdata/` for JSON fixtures and mock responses
   - Isolated component tests with mock navigation

6. **Legacy Integration**
   - `legacy/gui.go` preserved as behavior reference
   - Migration markers in code:
   ```go
   // LEGACY REFERENCE: gui.go lines 245-310
   // MIGRATION TARGET: components/chat/streaming.go
   ```

7. **Performance-Critical Areas**
   - Navigation stack with mutex protection
   - Lazy loading decorators for chat history
   - Streaming response buffer pools

### Migration Workflow

1. **Phase 1: Navigation Foundation**
   - Implement `navigation/stack.go`
   - Build test coverage for edge cases
   - Integrate with `app.go`

2. **Phase 2: Component Extraction**
   ```bash
   components/
   ├── sidebar/    # First - simplest dependency
   ├── input/      # Second - input processing
   └── chat/       # Last - complex with streaming
   ```

3. **Phase 3: Service Integration**
   - Implement repository pattern for storage
   - Build provider abstraction for AI
   - Add config encryption

4. **Phase 4: Polish & Optimization**
   - Dynamic resizing
   - Streaming performance
   - Animation system

### Critical Path Components
1. `navigation/stack.go` - Core navigation engine
2. `app/state.go` - Global state coordinator
3. `components/chat/streaming.go` - Performance hotspot
4. `services/ai/streaming.go` - AI integration point
5. `components/modals/manager.go` - User flow control

This structure maintains a clear boundary between:
- Core application logic
- State management
- UI presentation
- External services

All components communicate exclusively through the navigation stack and defined interfaces, ensuring testability and modularity while preserving the behavior specified in `legacy/gui.go`.