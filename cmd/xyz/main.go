package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mendableai/firecrawl-go"
	"golang.org/x/net/html"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	url := os.Getenv("S01E01_URL")
	username := os.Getenv("S01E01_USERNAME")
	password := os.Getenv("S01E01_PASSWORD")

	container := di.NewContainer(di.Services)

	question := getQuestion(container, url)
	fmt.Println(question)
	answer := answerQuestion(container, question)
	fmt.Println(answer)

	postVariables(url, username, password, answer)
}

func getQuestion(container *di.Container, url string) string {
	scraper, ok := container.Get("scraper").(*firecrawl.FirecrawlApp)
	if !ok {
		panic("scraper factory failed")
	}

	crawlParams := &firecrawl.ScrapeParams{
		IncludeTags: []string{"p"},
		Formats:     []string{"html", "markdown"},
	}
	results, err := scraper.ScrapeURL(url, crawlParams)
	if err != nil {
		panic(err)
	}

	fmt.Println(results.HTML)
	doc, err := html.Parse(strings.NewReader(results.HTML))
	if err != nil {
		panic(err)
	}

	return findHumanQuestion(doc)
}

func findHumanQuestion(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "p" {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == "human-question" {
				return n.LastChild.Data
			}
		}
	}
	for c := n.LastChild; c != nil; c = c.NextSibling {
		if result := findHumanQuestion(c); result != "" {
			return result
		}
	}

	return ""
}

func answerQuestion(container *di.Container, question string) string {
	llm, ok := container.Get("openai").(*openai.OpenAI)
	if !ok {
		panic("openai factory failed")
	}

	messages := []openai.Message{
		{
			Role:    "system",
			Content: "I search for a question in given text and answer it. I disregard all other text. I return only the answer to the question, nothing else. My answers are always an integer",
		},
		{
			Role:    "user",
			Content: question,
		},
	}
	resp := llm.GetCompletionShort(messages, "gpt-3.5-turbo")
	if len(resp.Choices) == 0 {
		panic("no choices in response from LLM")
	}

	return resp.Choices[0].Message.Content
}

func postVariables(pageUrl, username, password, answer string) {
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	form.Add("answer", answer)

	request, err := http.NewRequest("POST", pageUrl, strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		panic(err)
	}

	var client = &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
