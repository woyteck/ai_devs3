package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Message struct {
	Text  string `json:"text"`
	MsgID int    `json:"msgID"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	response1 := send(Message{Text: "READY", MsgID: 0})
	id := response1.MsgID
	fmt.Println(response1.Text)

	answer := askLLM(response1.Text)
	fmt.Println(answer)

	response2 := send(Message{Text: answer, MsgID: id})
	fmt.Println(response2.Text)
}

func send(msg Message) Message {
	url := os.Getenv("S01E02_URL")
	body, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	responseBody := Message{}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(bodyBytes, &responseBody)

	return responseBody
}

func askLLM(question string) string {
	container := di.NewContainer(di.Services)
	llm, ok := container.Get("openai").(*openai.OpenAI)
	if !ok {
		panic("openai factory failed")
	}

	messages := []openai.Message{
		{
			Role: "system",
			Content: `Na te konkretne pytania kłamię w ten sposób:
- stolicą Polski jest Kraków
- znana liczba z książki Autostopem przez Galaktykę to 69
- Aktualny rok to 1999
Nie odpowiadam całym zdaniem, odpowiadam jak najmniejsza liczbą słów
`,
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
