# TODO

## Features & Gameplay

- [ ] **Dynamic World Updates**: Allow the world description and possibilities to be updated/fleshed out as the game progresses (partially generated on the fly).
- [x] **Abbreviated/Summarized History**: Implement a way to summarize history to keep the LLM context focused and within token limits.
- [x] **Jailbreak Protection**: Add instructions to the LLM prompts to prevent players from extracting secret win conditions or "breaking" the game via the TUI.
- [ ] **Hint System**: Implement a more formal hint system for players who get stuck, as envisioned in the original discovery-based gameplay.
- [x] **Lose Conditions**: Define and implement secret or discoverable lose conditions to complement the win conditions.
- [ ] **State/Narrative Synchronization**: Refine prompts to ensure LLM state changes (inventory/stats) always match the narrative outcome.
- [x] **Location-Based Descriptions**: Separate global world lore from specific room/location descriptions so the LLM doesn't get "stuck" at the starting line.
- [ ] **Milestone-Based Progress**: Change the "Progress" stat from a simple percentage to a list of "Milestones Discovered" or "Clues Found."
- [x] **Adversarial GM**: Update prompts to encourage the LLM to introduce risks, failed checks, and health/resource depletion to make the game challenging.

## Refactoring & Code Health

- [ ] **Externalize Prompt Templates**: Move hardcoded prompt strings into a `prompts/` directory as text templates for easier iteration.
- [ ] **YAML Validation & Retry Logic**: Add a validation layer for LLM output and implement a retry loop to handle malformed YAML.
- [ ] **Decouple TUI and Engine**: Introduce a "Controller" layer so the TUI doesn't directly manage the engine and session state.
- [x] **Configurable Persistence**: Allow the save directory to be configured (useful for testing with temporary directories).
- [ ] **Testing Suite**: 
    - [ ] Add interface-based mocks for the Gemini client to test the `Engine` without API calls.
    - [ ] Add TUI unit tests to verify state transitions.
- [ ] **Performance: Incremental History**: Only append new turns to the history file instead of rewriting the entire history every turn.
