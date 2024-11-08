package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Test struct {
	Q string `json:"q,omitempty"`
	A string `json:"a,omitempty"`
}

type Item struct {
	Question string `json:"question"`
	Answer   int    `json:"answer"`
	Test     *Test  `json:"test,omitempty"`
}

type Message struct {
	ApiKey      string `json:"apikey"`
	Description string `json:"description"`
	Copyright   string `json:"copyright"`
	TestData    []Item `json:"test-data"`
}

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

	baseUrl := os.Getenv("CENTRALA_BASEURL")
	answerUrl := os.Getenv("S01E03_URL")
	key := os.Getenv("AI_DEVS_KEY")

	url := fmt.Sprintf("%s/data/%s/json.txt", baseUrl, key)
	message := fetchJson(url)
	corrected := correct(message, key)

	sendAnswer(key, answerUrl, corrected)
}

func sendAnswer(key, url string, message Message) {
	req := Answer{
		Task:   "JSON",
		ApiKey: key,
		Answer: message,
	}

	json, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func correct(message *Message, key string) Message {
	container := di.NewContainer(di.Services)
	llm, ok := container.Get("openai").(*openai.OpenAI)
	if !ok {
		panic("openai factory failed")
	}

	corrected := Message{}
	corrected.ApiKey = key
	corrected.Copyright = message.Copyright
	corrected.Description = message.Description
	corrected.TestData = []Item{}

	for _, i := range message.TestData {
		split := strings.Split(i.Question, "+")
		a, err := strconv.Atoi(strings.Trim(split[0], " "))
		if err != nil {
			panic(err)
		}
		b, err := strconv.Atoi(strings.Trim(split[1], " "))
		if err != nil {
			panic(err)
		}

		item := Item{
			Question: i.Question,
			Answer:   i.Answer,
		}
		if a+b != i.Answer {
			item.Answer = a + b
		}
		if i.Test != nil {
			correctedTest := Test{}
			correctedTest.Q = i.Test.Q
			correctedTest.A = answerQuestion(llm, i.Test.Q)
			item.Test = &correctedTest
		}

		corrected.TestData = append(corrected.TestData, item)
	}

	return corrected
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

func answerQuestion(llm *openai.OpenAI, question string) string {
	messages := []openai.Message{
		{
			Role:    "system",
			Content: "I return only the answer to the question, nothing else. My answers are very concise",
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
