package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/llama"
)

type Answer struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer any    `json:"answer"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	container := di.NewContainer(di.Services)
	llm, ok := container.Get("llama").(*llama.Llama)
	if !ok {
		panic("llama factory failed")
	}

	baseUrl := os.Getenv("CENTRALA_BASEURL")
	key := os.Getenv("AI_DEVS_KEY")
	url := fmt.Sprintf("%s/data/%s/cenzura.txt", baseUrl, key)

	text := fetchInputText(url)
	fmt.Println(text)

	context := `In order to prevent disclosing sensitive information I list all sensitive information.
I will use this information to replace it with this exact string: CENZURA.
I don't change the formatting of the text in any way. I do not add any new text.
Information considered sensitive:
- firstname
- surname
- city
- street name with a number
- person's age
I always include the resulting text only
`

	request := llama.CompletionRequest{
		Model:  "llama3:8b",
		Prompt: text,
		Stream: false,
		System: context,
	}

	resp := llm.GetCompletion(request)
	answer := strings.ReplaceAll(resp.Response, "CENZURA CENZURA", "CENZURA")

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(answer, "CENZURA")
}

func fetchInputText(url string) string {
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	return string(bytes)
}
