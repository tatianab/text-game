# Classic text-based game generator

I want to create a text-based game generator. 

##  UI
A terminal UI (tui) using the Go BubbleTea framework.
(https://github.com/charmbracelet/bubbletea).

## General Gameplay

The text-based game should be the classic style where the player has
to "discover" how to play and how to win, but there can be hints along
the way.

When we start the game, the user is prompted to either give
a hint about what sort of game they want to play, or pick "random".

The game engine will then generate a text based game for the player
to play, then provide prompts for the user to interact with to play the
game.

The game engine will use an LLM (Gemini) to parse the player's responses
as natural language. The player will provide their own API key.

## Gameplay details

### Game prompt

The game engine will use the user prompt as a rough starting point
for how the game should look, but due to randomness, the same prompt
could lead to wildly different games.

### Generation

All the "world" and "state" files will be stored as YAML.

When the game is first generated, the game engine will generate a
"world document", in a file, for the game, including a description of
the world, what is possible (e.g. what sorts of actions a player can
take), what sort of state will be held (inventory? relationships? hit
points?) and a general idea of what the win conditions are.

As the game progresses, these details will be fleshed out, and
the world document can be updated. It should
be exciting because the game is partially generated on the fly.

There should also be a file (or folder) containing the current state
of the game (what's in the players inventory? what's their health level?
how far have they progressed toward the goal(s)?).

Also, there should be a file (or folder) containing an abbreviated history
of what has happened so far in a game.

### Saving a game

The user should be able to save their game and resume it later, generate
a new game, and then go back to their old game.

## Security

The user's API key MUST be kept safe and not leaked.

This is a prototype, so it's OK if game files are stored in such a
way that they could be modified / read by the user. However, it should
be hard for the user to "jailbreak" the game from the UI. For example,
they shouldn't be able to ask for the secret win conditions and get
them back. However, it's OK if they could go to the game files and
find the secret win conditions.


