package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web <url>",
	Short: "Fetch and summarize web content",
	Long:  `This command fetches the content of a web page, summarizes it, and saves it as a markdown file.`,
	Args:  cobra.ExactArgs(1),
	Run:   runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
}

func runWeb(cmd *cobra.Command, args []string) {
	url := args[0]
	content, err := fetchWebContent(url)
	if err != nil {
		fmt.Printf("Error fetching web content: %v\n", err)
		return
	}

	summary, err := summarizeContent(content)
	if err != nil {
		fmt.Printf("Error summarizing content: %v\n", err)
		return
	}

	err = saveToMarkdown(url, summary)
	if err != nil {
		fmt.Printf("Error saving markdown: %v\n", err)
		return
	}

	fmt.Println("Web content summarized and saved successfully.")
}

func fetchWebContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func summarizeContent(content string) (string, error) {
	client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))

	resp, err := client.CreateCompletion(anthropic.CompletionRequest{
		Model:     "claude-2",
		Prompt:    fmt.Sprintf("Human: Summarize the following web content in markdown format:\n\n%s\n\nAssistant:", content),
		MaxTokens: 1000,
	})
	if err != nil {
		return "", err
	}

	return resp.Completion, nil
}

func saveToMarkdown(url string, content string) error {
	pageName := strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
	pageName = strings.ReplaceAll(pageName, "/", "-")
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s-web-rollup-%s.md", pageName, timestamp)

	return ioutil.WriteFile(filename, []byte(content), 0644)
}
