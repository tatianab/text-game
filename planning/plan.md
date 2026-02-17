# Objective
Build a text-based game generator using Go, BubbleTea, and Gemini. The game will generate unique worlds based on user hints, maintain state in YAML files, and use natural language processing for player actions.

# Key Files & Context
- `cmd/text-game/main.go`: Entry point for both `go install` and local development.
- `internal/engine`: Logic for LLM interaction and game state management.
- `internal/models`: YAML-serializable structs for World, State, and History.
- `internal/tui`: BubbleTea models and views.
- `go.mod`: Project dependencies.

# Implementation Steps
1. **Dependency Management**: Initialize and add dependencies:
    - `github.com/charmbracelet/bubbletea`
    - `github.com/charmbracelet/lipgloss`
    - `gopkg.in/yaml.v3`
    - `github.com/google/generative-ai-go/genai`
2. **Configuration & Secrets**: 
    - Implement an API key loader that reads from an environment variable (e.g., `GEMINI_API_KEY`).
3. **Data Modeling**: 
    - Define `World` (desc, rules, win conditions).
    - Define `GameState` (inventory, stats, current location).
    - Define `GameHistory` (list of actions and outcomes).
4. **Gemini Client**:
    - Implement a client in `internal/engine`.
    - Create prompt templates for:
        - `GenerateWorld(hint)`: Creates the initial YAML world and state.
        - `ProcessTurn(action, state, history)`: Returns narration and state updates.
5. **TUI Development (BubbleTea)**:
    - **Startup View**: Prompt user for a hint or "random".
    - **Loading View**: Display while waiting for the first generation.
    - **Main View**: 
        - Multi-line text area for the game log.
        - Text input for player commands.
        - Status bar (optional).
6. **Game Engine Logic**:
    - Logic to parse LLM response and update the local YAML files.
    - Logic to handle "saving" (copying current YAMLs to a named save slot).
7. **Persistence Layer**:
    - Functions to read/write `World`, `State`, and `History` to `.yaml` files in a `data/` or `saves/` directory.

# Verification & Testing
- **Unit Tests**: Test YAML marshalling/unmarshalling in `internal/models`.
- **Integration Tests**: Mock Gemini API to verify the engine's state update logic.
- **Manual Testing**: 
    - Run the TUI and verify navigation between screens.
    - Verify that YAML files are created and updated correctly during gameplay.
