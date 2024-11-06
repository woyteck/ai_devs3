package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	container := di.NewContainer(di.Services)

	llm, ok := container.Get("openai").(*openai.OpenAI)
	if !ok {
		panic("openai factory failed")
	}

	messages := []openai.Message{
		{
			Role:    "system",
			Content: "jestem John",
		},
		{
			Role:    "user",
			Content: "Hi",
		},
	}
	resp := llm.GetCompletionShort(messages, "gpt-3.5-turbo")
	fmt.Println(resp.Choices[0].Message.Content)
}
