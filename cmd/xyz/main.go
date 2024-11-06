package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/mendableai/firecrawl-go"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	container := di.NewContainer(di.Services)

	pageContents := scrapePage(container)
	fmt.Println(pageContents)
	answer := answerQuestion(container, pageContents)
	fmt.Println(answer)
}

func scrapePage(container *di.Container) string {
	scraper, ok := container.Get("scraper").(*firecrawl.FirecrawlApp)
	if !ok {
		panic("scraper factory failed")
	}

	crawlParams := &firecrawl.ScrapeParams{
		IncludeTags: []string{"p"},
	}
	results, err := scraper.ScrapeURL("https://xyz.ag3nts.org/", crawlParams)
	if err != nil {
		panic(err)
	}

	return results.Markdown
}

func answerQuestion(container *di.Container, question string) string {
	llm, ok := container.Get("openai").(*openai.OpenAI)
	if !ok {
		panic("openai factory failed")
	}

	messages := []openai.Message{
		{
			Role:    "system",
			Content: "I search for a question in given text and answer it. I disregard all other text. I return only the answer to the question, nothing else. My answers are concise.",
		},
		{
			Role:    "user",
			Content: question,
		},
	}
	resp := llm.GetCompletionShort(messages, "gpt-3.5-turbo")
	return resp.Choices[0].Message.Content
}
