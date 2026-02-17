# text-game
Text-based game generator

(IN PROGRESS)

## Play the game

1.  **Install the game**:
    ```bash
    go install github.com/tatianab/text-game/cmd/text-game@latest
    ```
    *Ensure your `GOBIN` (usually `~/go/bin`) is in your `PATH`.*

2.  **Set your Gemini API key**: 
    Obtain a free API key from [Google AI Studio](https://aistudio.google.com/app/apikey) and set it in your environment:
    ```bash
    export GEMINI_API_KEY=your_api_key_here
    ```

3.  **Run the game**:
    ```bash
    text-game
    ```

## Development

If you have cloned the repository, you can run the game directly:
```bash
go run main.go
```
