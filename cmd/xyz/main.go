package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/openai"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	key := os.Getenv("OPENAI_API_KEY")
	llm := openai.NewOpenAI(key)

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
