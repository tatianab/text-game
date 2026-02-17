# TODO

- [ ] **Dynamic World Updates**: Allow the world description and possibilities to be updated/fleshed out as the game progresses (partially generated on the fly).
- [ ] **Abbreviated/Summarized History**: Implement a way to summarize history to keep the LLM context focused and within token limits.
- [ ] **Jailbreak Protection**: Add instructions to the LLM prompts to prevent players from extracting secret win conditions or "breaking" the game via the TUI.
- [ ] **Hint System**: Implement a more formal hint system for players who get stuck, as envisioned in the original discovery-based gameplay.
- [ ] **Lose Conditions**: Define and implement secret or discoverable lose conditions to complement the win conditions.
- [ ] **State/Narrative Synchronization**: Refine prompts to ensure LLM state changes (inventory/stats) always match the narrative outcome.
- [ ] **Location-Based Descriptions**: Separate global world lore from specific room/location descriptions so the LLM doesn't get "stuck" at the starting line.
- [ ] **Milestone-Based Progress**: Change the "Progress" stat from a simple percentage to a list of "Milestones Discovered" or "Clues Found."
- [ ] **Adversarial GM**: Update prompts to encourage the LLM to introduce risks, failed checks, and health/resource depletion to make the game challenging.
