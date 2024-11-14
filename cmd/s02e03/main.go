package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Message struct {
	Description string `json:"description"`
}

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

	url := fmt.Sprintf("%s/data/%s/robotid.json", os.Getenv("CENTRALA_BASEURL"), os.Getenv("AI_DEVS_KEY"))
	message := fetchJson(url)
	fmt.Println(message.Description)

	refinedDescription := refineDescription(llm, message.Description)
	fmt.Println(refinedDescription)

	imageUrl := generateImage(llm, refinedDescription)
	fmt.Println(imageUrl)

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(imageUrl, "robotid")
}

func fetchJson(url string) *Message {
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	j := &Message{}
	err = json.Unmarshal(bytes, j)
	if err != nil {
		panic(err)
	}

	return j
}

func refineDescription(llm *openai.OpenAI, description string) string {
	messages := []openai.Message{
		{
			Role:    "system",
			Content: "Wyciągam opis przedmitu z podanego tekstu",
		},
		{
			Role:    "user",
			Content: description,
		},
	}
	completions := llm.GetCompletionShort(messages, "gpt-4")
	if len(completions.Choices) == 0 {
		panic("no completions returned")
	}

	return completions.Choices[0].Message.Content
}

func generateImage(llm *openai.OpenAI, prompt string) string {
	result := llm.CreateImageShort(prompt)
	if len(result.Data) == 0 {
		panic("no images generated")
	}

	return result.Data[0].Url
}
