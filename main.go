package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	path "path/filepath"
	"time"

	"github.com/spf13/cobra"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

var filePath string
var rootCmd = &cobra.Command{
	Use:   "aira",
	Short: "Aira CLI",
	Long:  "Aira CLI is a tool for interacting with the Aira API",
	Run: func(cmd *cobra.Command, args []string) {
		content, err := readFile(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}

		sendToAIReview(content)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "File to read")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func readFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("no file provided")
	}

	absolutePath, err := path.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("error resolving path: %v", err)
	}

	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return "", fmt.Errorf("error reading file %s: %v", filePath, err)
	}
	return string(content), nil
}

func renderSpinner(done chan bool) {
	spinner := []rune{'|', '/', '-', '\\'}
	i := 0
	for {
		select {
		case <-done: // Se receber um sinal no canal 'done', a função encerra.
			return
		default:
			// O truque aqui é o '\r' (carriage return). Ele move o cursor
			// para o início da linha sem pular para a próxima,
			// permitindo que o próximo caractere sobrescreva o anterior.
			fmt.Printf("\rAnalisando com o Ollama... %c ", spinner[i])
			i = (i + 1) % len(spinner) // Avança para o próximo caractere do spinner
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func sendToAIReview(content string) {
	const url = "http://localhost:11434/api/generate"
	const model = "llama3"

	prompt := fmt.Sprintf(`
  Act as a seasoned Principal Software Engineer reviewing a pull request (PR) from a colleague. Your goal is to provide a comprehensive, constructive, and actionable code review.

  Analyze the following code snippet with these key areas in mind:

  1.  **Bugs and Edge Cases:** Identify potential bugs, logical errors, race conditions, or unhandled edge cases. Think about what could go wrong in a production environment.
  2.  **Best Practices & Idiomatic Code:** Does the code adhere to the language's best practices and idiomatic conventions? Is it clean and easy to understand?
  3.  **Performance and Efficiency:** Look for performance bottlenecks, inefficient algorithms, or unnecessary resource consumption (memory, CPU).
  4.  **Readability & Maintainability:** Assess the code's clarity and long-term maintainability. Suggest improvements to variable names, comments, and overall structure.
  5.  **Refactoring Opportunities:** Propose specific refactorings that could improve the code's architecture, reduce complexity, or eliminate redundant code (DRY - Don't Repeat Yourself).
  6.  **Security:** Point out any potential security vulnerabilities.

  Please structure your review in Markdown. Start with a brief summary, then use clear headings for each point of your feedback.

  Here is the code to be reviewed:
  -------------------------------
  %s
  `, content)

	requestData := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Fatalf("error converting data to JSON: %v", err)
	}

	fmt.Println("🤖 Sending code to Ollama... Please wait.")
	done := make(chan bool)
	go renderSpinner(done)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	done <- true
	if err != nil {
		log.Fatalf("error making request to Ollama: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading response from Ollama: %v", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		log.Fatalf("error unmarshalling response from Ollama: %v", err)
	}

	fmt.Println("\n--- ✅ Code Review Received ---")
	fmt.Println(ollamaResp.Response)

}
