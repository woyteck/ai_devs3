package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Image struct {
	Url     string
	Caption string
}

type ScrapeResults struct {
	Paragraphs []string
	Images     []Image
	Audio      []string
}

type Question struct {
	Index int
	Text  string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables instead")
	}

	// container := di.NewContainer(di.Services)
	// llm, ok := container.Get("openai").(*openai.OpenAI)
	// if !ok {
	// 	panic("openai factory failed")
	// }

	// cache, ok := container.Get("redis").(*redis.Client)
	// if !ok {
	// 	panic("openai factory failed")
	// }

	// url := fmt.Sprintf("%s/dane/arxiv-draft.html", os.Getenv("CENTRALA_BASEURL"))
	// results := scrapePage(url)
	// normalized := normalizeData(llm, cache, results)
	// fmt.Println(normalized)

	questions := fetchQuestions()
	fmt.Println(questions)
}

func fetchQuestions() []Question {
	url := fmt.Sprintf("%s/data/%s/arxiv.txt", os.Getenv("CENTRALA_BASEURL"), os.Getenv("AI_DEVS_KEY"))

	response, err := http.Get(url)
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(contents))

	questions := []Question{}

	return questions
}

func normalizeData(llm *openai.OpenAI, cache *redis.Client, data ScrapeResults) []string {
	results := []string{}
	for _, text := range data.Paragraphs {
		results = append(results, text)
	}

	for _, audio := range data.Audio {
		transcript := transcriptAudio(llm, cache, audio)
		results = append(results, transcript)
	}

	for _, image := range data.Images {
		description := describeImage(llm, cache, image)
		results = append(results, description)
	}

	return results
}

func describeImage(llm *openai.OpenAI, cache *redis.Client, image Image) string {
	ctx := context.Background()

	imageUrl := fmt.Sprintf("%s/dane/%s", os.Getenv("CENTRALA_BASEURL"), image.Url)

	var description string
	description, err := cache.Get(ctx, imageUrl).Result()
	if err != nil {
		fmt.Println("cache miss")

		imageResponse, err := http.Get(imageUrl)
		fileContents, err := io.ReadAll(imageResponse.Body)
		if err != nil {
			panic(err)
		}

		encodedImage := base64.StdEncoding.EncodeToString(fileContents)
		messages := []openai.ImageMessage{
			{
				Role: "system",
				Content: []openai.Content{
					{
						Type: "text",
						Text: "I describe what's on the image.",
					},
				},
			},
			{
				Role: "user",
				Content: []openai.Content{
					{
						Type: "image_url",
						ImageURL: openai.ImageURL{
							URL: "data:image/png;base64," + encodedImage,
						},
					},
					{
						Type: "text",
						Text: image.Caption,
					},
				},
			},
		}
		completions := llm.GetImageCompletionShort(messages, "gpt-4-turbo")

		if len(completions.Choices) == 0 {
			panic("no completions returned by LLM")
		}

		description = completions.Choices[0].Message.Content

		cache.Set(ctx, imageUrl, description, time.Hour)
	} else {
		fmt.Println("cache hit")
	}

	return description
}

func transcriptAudio(llm *openai.OpenAI, cache *redis.Client, url string) string {
	ctx := context.Background()

	audioUrl := fmt.Sprintf("%s/dane/%s", os.Getenv("CENTRALA_BASEURL"), url)

	var transcript string
	transcript, err := cache.Get(ctx, audioUrl).Result()
	if err != nil {
		fmt.Println("cache miss")

		audioResponse, err := http.Get(audioUrl)
		fileContents, err := io.ReadAll(audioResponse.Body)
		if err != nil {
			panic(err)
		}

		transcript = llm.GetTranscription(fileContents, "whisper-1", "mp3")

		cache.Set(ctx, audioUrl, transcript, time.Hour)
	} else {
		fmt.Println("cache hit")
	}

	return transcript
}

func scrapePage(url string) ScrapeResults {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Non-200 response: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	results := ScrapeResults{}

	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		results.Paragraphs = append(results.Paragraphs, text)
	})

	doc.Find("figure").Each(func(i int, s *goquery.Selection) {
		image := Image{}

		img := s.Find("img")
		if src, exists := img.Attr("src"); exists {
			image.Url = src
		}

		caption := s.Find("figcaption").Text()
		image.Caption = caption

		results.Images = append(results.Images, image)
	})

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists && strings.Contains(link, ".mp3") {
			results.Audio = append(results.Audio, link)
		}
	})

	return results
}
