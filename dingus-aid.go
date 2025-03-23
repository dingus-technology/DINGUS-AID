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
	"strings"
)

const sessionFile = "cli_session_history.json"
const configFile = "config.json"

// Structure to hold session history
type Session struct {
	History []Entry `json:"history"`
}

type Entry struct {
	Query    string `json:"query"`
	Response string `json:"response"`
}

var openaiAPIKey string

// Load session data from file
func loadSession() (Session, error) {
	var session Session
	if _, err := os.Stat(sessionFile); err == nil {
		data, err := os.ReadFile(sessionFile)
		if err != nil {
			return session, err
		}
		err = json.Unmarshal(data, &session)
		if err != nil {
			return session, err
		}
	}
	return session, nil
}

// Save session data to file
func saveSession(session Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionFile, data, 0644)
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

// Get command suggestion from OpenAI API
func getCommandSuggestion(query string, history []Entry) (string, error) {
	context := ""
	for _, entry := range history {
		context += fmt.Sprintf("User: %s\nBot: %s\n", entry.Query, entry.Response)
	}

	prompt := fmt.Sprintf(`
Here is the history of the current CLI session:

<HISTORY> %s </HISTORY>

ONLY respond with the command that should be run next.
The command must be a valid command.

Now, given the following query, suggest the correct command:

User: %s

Suggested command:`, context, query)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []interface{}{
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

// Main function
func main() {
	// Check if query argument is provided
	if len(os.Args) < 2 {
		log.Fatal("Please provide a query as a command-line argument")
	}
	query := strings.Join(os.Args[1:], " ")

	// Try loading API key from config file
	var err error
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

	// Load session history
	session, err := loadSession()
	if err != nil {
		log.Fatalf("Error loading session: %v", err)
	}

	// Get the suggested command from OpenAI
	suggestedCommand, err := getCommandSuggestion(query, session.History)
	if err != nil {
		log.Fatalf("Error getting command suggestion: %v", err)
	}

	// Output the suggested command
	fmt.Printf("Suggested command: %s\n", suggestedCommand)

	// Ask if the user wants to run the command
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to run this command? (y/n): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading confirmation: %v", err)
	}
	confirm = strings.TrimSpace(confirm)

	if strings.EqualFold(confirm, "y") {
		// Run the suggested command
		output, err := runCommand(suggestedCommand)
		if err != nil {
			fmt.Printf("Command returned error: %v\n", err)
			fmt.Printf("Output:\n%s\n", output)
		} else {
			// Output the result
			fmt.Printf("Command output:\n%s\n", output)
		}

		// Update session history
		session.History = append(session.History, Entry{
			Query:    query,
			Response: output,
		})

		// Save updated session
		err = saveSession(session)
		if err != nil {
			log.Fatalf("Error saving session: %v", err)
		}
	} else {
		fmt.Println("Command not executed.")
	}
}