package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Config files stored in user's home directory
var (
	configDir     string
	configFile    string
	openaiAPIKey  string
)

// Command history tracking (in-memory)
type CommandHistory struct {
	Entries  []HistoryEntry
	MaxSize  int
	MaxWords int
}

type HistoryEntry struct {
	Command string
	Output  string
}

// Create a global history tracker
var history = CommandHistory{
	Entries:  []HistoryEntry{},
	MaxSize:  8,  // Store the last 5 commands
	MaxWords: 160, // Limit to last 100 words per entry
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorPurple = "\033[35m"
	colorBold   = "\033[1m"
)

// API cost rates per million tokens
const (
	inputTokenCost  = 0.15  // $0.15 per million tokens
	outputTokenCost = 0.60  // $0.60 per million tokens
)

// Initialize config directory and files
func initConfigFiles() error {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}
	
	// Create .dingus-aid directory in user's home
	configDir = filepath.Join(homeDir, ".dingus-aid")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	
	// Set global file paths
	configFile = filepath.Join(configDir, "config.json")
	
	return nil
}

// Add command and its output to history
func (h *CommandHistory) Add(command, output string) {
	// Trim output to max words
	words := strings.Fields(output)
	if len(words) > h.MaxWords {
		words = words[len(words)-h.MaxWords:]
		output = strings.Join(words, " ")
	}
	
	// Create new entry
	entry := HistoryEntry{
		Command: command,
		Output:  output,
	}
	
	// Add to history, keeping only the most recent MaxSize entries
	h.Entries = append(h.Entries, entry)
	if len(h.Entries) > h.MaxSize {
		h.Entries = h.Entries[len(h.Entries)-h.MaxSize:]
	}
}

// Get history context as formatted string for the prompt
func (h *CommandHistory) GetContext() string {
	if len(h.Entries) == 0 {
		return ""
	}
	
	var context strings.Builder
	context.WriteString("\n\nRecent command history (for context):\n")
	
	for i, entry := range h.Entries {
		context.WriteString(fmt.Sprintf("\nCOMMAND %d: %s\nOUTPUT %d: %s\n", 
			i+1, entry.Command, i+1, entry.Output))
	}
	
	return context.String()
}

// Save API key to a configuration file
func saveAPIKey(apiKey string) error {
	configData := map[string]string{
		"OPENAI_API_KEY": apiKey,
	}
	configJSON, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, configJSON, 0600)
}

// Load API key from configuration file
func loadAPIKey() (string, error) {
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return "", err
		}
		var configData map[string]string
		err = json.Unmarshal(data, &configData)
		if err != nil {
			return "", err
		}
		if apiKey, exists := configData["OPENAI_API_KEY"]; exists {
			return apiKey, nil
		}
	}
	return "", fmt.Errorf("API key not found")
}

// Remove all configuration files
func cleanupConfigFiles() error {
	// Remove the entire config directory
	err := os.RemoveAll(configDir)
	if err != nil {
		return fmt.Errorf("failed to remove config files: %v", err)
	}
	return nil
}

// Get command suggestion from OpenAI API and return token usage
func getCommandSuggestion(query string) (string, int, int, error) {
	// Add command history context to the prompt
	historyContext := history.GetContext()
	
	prompt := fmt.Sprintf(`
Always adhere to these rules when suggesting the command:
- The command must be a valid terminal command.
- It should be relevant to the user's query.
- Continue the conversation by giving useful commands.
- Consider the chat history and make the command more useful than before based on the user's follow up questions.
- Use information from the chat history to help generate the command.
- The command should not require user input.
- It must not be destructive or modify the system in any harmful way.
- The command should not require additional software, configuration, or access to external resources, the internet, or sensitive information.

Format your response as follows:
- Only respond with the suggested command.
- Ensure the command is executable in the current session.
- Do not include any additional information or context.
- Do not include any formattings.
- Do not include 'dingus-aid' in the command.

The command line history is as follows:

<COMMAND_HISTORY> %s </COMMAND_HISTORY>

The user query is as follows:

<USER_QUESTION> %s </USER_QUESTION>

Suggested command:`, historyContext, query)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "You are a helpful assistant designed to suggest valid, safe, and relevant terminal commands based on user input."},
			map[string]interface{}{"role": "user", "content": prompt},
		},
		"max_tokens": 100,
	}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, 0, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqData))
	if err != nil {
		return "", 0, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", 0, 0, fmt.Errorf("error from OpenAI API: %s - %s", resp.Status, string(bodyBytes))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", 0, 0, err
	}

	// Extract token usage
	promptTokens, completionTokens := 0, 0
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			promptTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			completionTokens = int(ct)
		}
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if text, ok := message["content"].(string); ok {
					return strings.TrimSpace(text), promptTokens, completionTokens, nil
				}
			}
		}
	}

	return "", promptTokens, completionTokens, fmt.Errorf("no valid response from OpenAI API")
}

// Calculate API call cost
func calculateCost(promptTokens, completionTokens int) float64 {
	promptCost := float64(promptTokens) * inputTokenCost / 1_000_000
	completionCost := float64(completionTokens) * outputTokenCost / 1_000_000
	return promptCost + completionCost
}

// Run the suggested command
func runCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Copy text to clipboard based on OS
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("cmd", "/c", "echo", text, "|", "clip")
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported platform")
	}
	
	// For non-Windows platforms
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Main function
func main() {
	// Initialize config directory and files
	err := initConfigFiles()
	if err != nil {
		log.Fatalf("Error initialising config: %v", err)
	}

	// Check if this is a cleanup command
	if len(os.Args) >= 2 && os.Args[1] == "cleanup" {
		err := cleanupConfigFiles()
		if err != nil {
			log.Fatalf("Error cleaning up config files: %v", err)
		}
		fmt.Printf("%sConfiguration files removed successfully!%s\n", colorGreen, colorReset)
		return
	}

	// Check if query argument is provided
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  dingus-aid <query>     - Get command suggestion")
		fmt.Println("  dingus-aid cleanup     - Remove all configuration files")
		os.Exit(1)
	}
	
	// Join all arguments as the query except for the program name
	query := strings.Join(os.Args[1:], " ")

	// Try loading API key from config file
	openaiAPIKey, err = loadAPIKey()
	if err != nil || openaiAPIKey == "" {
		// If API key is not found or empty, ask user for it and save it
		fmt.Print("Enter your OpenAI API Key: ")
		reader := bufio.NewReader(os.Stdin)
		apiKey, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading API key: %v", err)
		}
		openaiAPIKey = strings.TrimSpace(apiKey)

		// Save the key to the configuration file
		err = saveAPIKey(openaiAPIKey)
		if err != nil {
			log.Fatalf("Error saving API key: %v", err)
		}
		fmt.Println("API key saved.")
	}

	// Get the suggested command from OpenAI and token usage
	suggestedCommand, promptTokens, completionTokens, err := getCommandSuggestion(query)
	if err != nil {
		log.Fatalf("Error getting command suggestion: %v", err)
	}

	// Calculate the cost
	cost := calculateCost(promptTokens, completionTokens)

	// Output the suggested command with decoration
	fmt.Printf("\n%s%s%sSuggested command:%s %s%s%s%s%s\n\n", 
		colorBold, colorYellow, colorBold, 
		colorReset,
		colorCyan, colorBold, 
		suggestedCommand,
		colorReset, colorReset)
		
	// Output the token usage and cost in purple
	fmt.Printf("%sQuery cost: $%.6f%s\n\n", colorPurple, cost, colorReset)

	// Ask if the user wants to run the command
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to run this command? (y/n/c - 'c' to copy to clipboard): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading confirmation: %v", err)
	}
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	var output string
	switch confirm {
	case "y":
		// Run the suggested command
		output, err = runCommand(suggestedCommand)
		if err != nil {
			fmt.Printf("Command returned error: %v\n", err)
			fmt.Printf("Output:\n%s\n", output)
		} else {
			// Output the result
			fmt.Printf("\n%sCommand output:%s\n%s\n", colorBold, colorReset, output)
		}
		
		// Add to command history
		history.Add(suggestedCommand, output)
		
	case "c":
		// copy to clipboard
		err = copyToClipboard(suggestedCommand)
		if err == nil {
			fmt.Printf("%sCommand copied to clipboard!%s\n\n", colorGreen, colorReset)
		}
		
		fmt.Println("Command not executed.")
	default:
		fmt.Println("Command not executed.")
	}
}