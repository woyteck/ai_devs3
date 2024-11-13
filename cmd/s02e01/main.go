package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"woyteck.pl/ai_devs3/internal/aidevs"
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

	ctx := context.Background()

	url := os.Getenv("S02E01_URL")

	zipFile := fetchZip(url)
	destination := "/tmp/archive.zip"

	f, err := os.Create(destination)
	if err != nil {
		panic(err)
	}

	_, err = f.Write(zipFile)
	if err != nil {
		panic(err)
	}

	archive, err := zip.OpenReader(destination)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	transcriptions := []string{}

	cache, ok := container.Get("redis").(*redis.Client)
	if !ok {
		panic("openai factory failed")
	}

	interrigations, err := cache.Get(ctx, "interrigations").Result()
	if err != nil {
		fmt.Println("cache miss")

		for _, f := range archive.File {
			fmt.Println(f.Name)
			file, err := f.Open()
			if err != nil {
				panic(err)
			}

			fileContents, err := io.ReadAll(file)
			if err != nil {
				panic(err)
			}

			transcription := llm.GetTranscription(fileContents, "whisper-1", "m4a")
			fmt.Println(transcription)
			transcriptions = append(transcriptions, transcription)
		}

		interrigations = strings.Join(transcriptions, "\n")

		cache.Set(ctx, "interrigations", interrigations, time.Hour)
	} else {
		fmt.Println("cache hit")
	}

	context := fmt.Sprintf(
		"%s\n\nTreści przesłuchań świadków:\n%s",
		"Jestem detektywem, prowadzę dochodzenie w sprawie Andrzeja Maja.\nAnalizuję fakty krok po kroku, używam dedukcji, żeby wyciągnąć wnioski.",
		interrigations,
	)

	messages := []openai.Message{
		{
			Role:    "system",
			Content: context,
		},
		{
			Role:    "user",
			Content: "Wywnioskuj z treści przesłuchań na jakiej uczelni pracował Andrzej Maj, a potem daj mi adres wydziału tej uczelni, w którym pracował. Zwróć tylko adres, nic więcej.",
		},
	}
	resp := llm.GetCompletionShort(messages, "gpt-4o")
	if len(resp.Choices) == 0 {
		panic("no choices in response from LLM")
	}

	fmt.Println(resp.Choices[0].Message.Content)

	messages = []openai.Message{
		{
			Role:    "system",
			Content: "Wyciągam nazwę ulicy z adresu. Zwracam tylko i wyłącznie nazwę ulicy.",
		},
		{
			Role:    "user",
			Content: resp.Choices[0].Message.Content,
		},
	}
	resp = llm.GetCompletionShort(messages, "gpt-3.5-turbo")
	if len(resp.Choices) == 0 {
		panic("no choices in response from LLM")
	}

	fmt.Println(resp.Choices[0].Message.Content)

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(resp.Choices[0].Message.Content, "mp3")
}

func fetchZip(url string) []byte {
	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	return bytes
}
