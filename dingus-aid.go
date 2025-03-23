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
	configDir    string
	configFile   string
	openaiAPIKey string
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Structure to hold a command entry
type Entry struct {
	Query    string
	Response string
}

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

// Truncate history context to approximately the last 100 words
func truncateHistoryContext(context string) string {
	words := strings.Fields(context)
	if len(words) <= 100 {
		return context
	}
	
	// Take only the last 100 words
	truncatedWords := words[len(words)-100:]
	return strings.Join(truncatedWords, " ")
}

// Get command suggestion from OpenAI API
func getCommandSuggestion(query string, history []Entry) (string, error) {
	// Build context from the last 5 entries in history (or fewer if not available)
	context := ""
	startIdx := 0
	if len(history) > 5 {
		startIdx = len(history) - 5
	}
	
	for i := startIdx; i < len(history); i++ {
		context += fmt.Sprintf("User: %s\nBot: %s\n", history[i].Query, history[i].Response)
	}
	
	// Truncate context to approximately the last 100 words
	context = truncateHistoryContext(context)

	prompt := fmt.Sprintf(`
Here is the history of the current CLI session:

<HISTORY> %s </HISTORY>

Always adhere to these rules when suggesting the command:
- The command must be a valid terminal command.
- It should be a continuation of the conversation.
- It must be relevant to the user's query.
- The command should not require user input.
- It must not be destructive or modify the system in any harmful way.
- Avoid duplicates of previous commands in this session.
- The command should not require additional software, configuration, or access to external resources, the internet, or sensitive information.
- The command should not reference user-specific files or data.

Format your response as follows:
- Only respond with the suggested command.
- Ensure the command is executable in the current session.
- Do not include any additional information or context.
- Do not include any formattings.

The user query is as follows:

<USER_QUESTION> %s </USER_QUESTION>

Suggested command:`, context, query)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "You are a helpful assistant designed to suggest valid, safe, and relevant terminal commands based on user input and session history."},
			map[string]interface{}{"role": "user", "content": prompt},
		},
		"max_tokens": 100,
	}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error from OpenAI API: %s - %s", resp.Status, string(bodyBytes))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if text, ok := message["content"].(string); ok {
					return strings.TrimSpace(text), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no valid response from OpenAI API")
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

// Copy and paste to terminal (simulates keyboard input)
func copyPasteToTerminal(text string) error {
	// Copy to clipboard first
	err := copyToClipboard(text)
	if err != nil {
		return err
	}
	
	// Simulate paste operation based on OS
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("osascript", "-e", 
			`tell application "System Events" to keystroke "v" using command down`)
	case "linux":
		// Using xdotool to simulate Ctrl+Shift+V (paste in terminal)
		cmd = exec.Command("xdotool", "key", "ctrl+shift+v")
	case "windows":
		// Windows terminal typically uses Ctrl+V
		cmd = exec.Command("cmd", "/c", "echo Paste operation not fully supported on Windows")
		fmt.Println("For Windows, please use Ctrl+V to paste manually.")
		return nil
	default:
		return fmt.Errorf("unsupported platform")
	}
	
	return cmd.Run()
}

// Main function
func main() {
	// Initialize config directory and files
	err := initConfigFiles()
	if err != nil {
		log.Fatalf("Error initialising config: %v", err)
	}

	// Create an in-memory history list
	history := []Entry{}
	
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

	// Get the suggested command from OpenAI
	suggestedCommand, err := getCommandSuggestion(query, history)
	if err != nil {
		log.Fatalf("Error getting command suggestion: %v", err)
	}

	// Output the suggested command with decoration
	fmt.Printf("\n%s%s%sSuggested command:%s %s%s%s%s%s\n\n", 
		colorBold, colorYellow, colorBold, 
		colorReset,
		colorCyan, colorBold, 
		suggestedCommand,
		colorReset, colorReset)

	// Copy command to clipboard automatically for convenience
	// err = copyToClipboard(suggestedCommand)
	// if err == nil {
	// 	fmt.Printf("%sCommand copied to clipboard!%s\n\n", colorGreen, colorReset)
	// }

	// Ask if the user wants to run the command
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to run this command? (y/n/c - 'c' to copy to clipboard): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading confirmation: %v", err)
	}
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	switch confirm {
	case "y":
		// Run the suggested command
		output, err := runCommand(suggestedCommand)
		if err != nil {
			fmt.Printf("Command returned error: %v\n", err)
			fmt.Printf("Output:\n%s\n", output)
		} else {
			// Output the result
			fmt.Printf("\n%sCommand output:%s\n%s\n", colorBold, colorReset, output)
		}

		// Update in-memory history
		// We'll keep this entry in memory for this session, but it won't persist
		history = append(history, Entry{
			Query:    query,
			Response: output,
		})
		
		// If we have more than 5 entries, remove the oldest one(s)
		for len(history) > 5 {
			history = history[1:]
		}
		
	case "c":
		// copy to clipboard
		err = copyToClipboard(suggestedCommand)
		if err == nil {
			fmt.Printf("%sCommand copied to clipboard!%s\n\n", colorGreen, colorReset)
		}
		// err = copyPasteToTerminal(suggestedCommand)
		// if err != nil {
		// 	fmt.Printf("Could not paste to terminal: %v\n", err)
		// 	fmt.Println("You may need to install xdotool (Linux) or ensure permissions are set correctly.")
		// 	fmt.Println("The command is still in your clipboard - you can manually paste it.")
		// } else {
		// 	fmt.Printf("%sCommand pasted to terminal!%s\n", colorGreen, colorReset)
		// }
		
		// Don't record anything in history as we didn't run the command
		fmt.Println("Command not executed.")
	default:
		fmt.Println("Command not executed.")
	}
}