# Never forget those pesky CLI commands again - [**Dingus Copilot**](http://www.dingusai.dev) 🧙‍♂️

<div style="text-align: center;">
   <img src="assets/dingus-demo.gif" alt="Demo" style="max-width: 70%; height: auto;">
</div>

Welcome to **Dingus Aid**! 🛠️ This command-line tool is here to help you by suggesting and running the right commands for you—automatically! Whether you're a beginner or a seasoned pro, Dingus Aid uses the power of AI to figure out the next step based on your query.

### What's This All About?
Do you find yourself stuck in the middle of a command-line labyrinth? No worries! This tool will analyze your query, give you a **suggested command**, and then let you run it with just a few keystrokes. It's like having a command-line buddy who never gets tired! 💬

---

## Quick Setup

Make sure you have docker installed.

1. **Create directories**
   ```bash
   mkdir output logs
   ```
2. **Build the Binary**
   - Let's set up the Dingus CLI tool by building the Docker image:
     ```bash
     ./dingus-copilot-installer.sh
     ```
     or
     ```bash
     bash dingus-copilot-installer.sh
     ```

3. **Install the Tool**
   - The `dingus-copilot` binary will be placed in `/usr/local/bin`, ready for use from anywhere in your terminal.

4. **Configure Your API Key**
   - When you run the tool for the first time, you'll need to enter your OpenAI API key. Don't worry, it'll be saved in the `config.json` file for later and easy to remove if you need to wipe it from your machine 🔑.

5. **Start Using It**
   - Set API key:
     ```bash
     dingus-copilot hi
     Enter your OpenAI API Key: <YOUR OPENAI API KEY HERE>
     ```
   - To ask Dingus Aid a question, just run:
     ```
     dingus-copilot How do I list files in a directory?
     ```
   - Dingus Aid will suggest the correct command and ask if you want to run it. Simple as that!


---

## Features

- **Command Suggestions**: Ask a question, and Dingus Aid will suggest the most appropriate command.
- **Automatic History Tracking**: It keeps track of your session history, so it remembers the context of your previous queries.
- **Run Commands with Confidence**: If you agree with the suggestion, Dingus Aid will run it for you and show the results.
- **Session Persistence**: Your queries and their results are saved in a history file, so you can pick up where you left off.

---

## Example Workflow

1. **Query the CLI Tool**:
   ```bash
   dingus-copilot How do I copy files?
   ```
   Dingus Aid might suggest:
   ```bash
   cp source_file destination_file
   ```

2. **Confirm the Command**:
   Dingus Aid will ask:
   ```bash
   Do you want to run this command? (y/n/c):
   ```
   Hit **y** to execute the command, or **n** to skip. **c** will copy the command to clipboard.

3. **Enjoy the Output**:
   Dingus Aid will show you the results of the command execution.

---

## Advanced Features

- **OpenAI Integration**: It uses the OpenAI API to generate intelligent command suggestions. This keeps it smart and adaptable to your workflow!

---

## Troubleshooting

- **API Key Missing**: If you don't have an API key, the tool will prompt you to enter one. Make sure you save it, and Dingus Aid will handle the rest!
  
- **Binary Not Found**: If you ever get a `dingus-copilot command not found` error, just run `bash dingus-copilot-installer.sh` again, and it will restore the binary.

---

## Contribute

Want to make Dingus Aid even smarter? 🧠 Feel free to fork this repo and create a pull request with your changes. If you've got a cool new feature in mind, let us know, and we'll make it happen!

---

## TODO

- Ingest the command line history into the prompt.
- Include extra context for the CLI, such as the current directory and its contents.
- Upgrade gracefulness of `dingus-copilot-installer.sh`.
- Output cli stdout in real time not once commands have all finshed running.

---

## Credits

This tool is powered by the [**Dingus**](http://www.dingusai.dev) team and **OpenAI's GPT models** for intelligent command suggestions.

Happy command-line adventuring! 🎉