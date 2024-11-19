package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Image struct {
	Url     string
	Caption string
}

type ScrapeResults struct {
	Sections []*Section
}

type Section struct {
	Title      string
	Paragraphs []string
	Images     []Image
	Audio      []string
}

type Question struct {
	Index  int
	Text   string
	Answer string
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

	cache, ok := container.Get("redis").(*redis.Client)
	if !ok {
		panic("openai factory failed")
	}

	url := fmt.Sprintf("%s/dane/arxiv-draft.html", os.Getenv("CENTRALA_BASEURL"))
	results := scrapePage(url)

	normalized := normalizeData(llm, cache, results)

	response := map[string]string{}
	for _, question := range fetchQuestions() {
		answer := answerQuestion(llm, question.Text, strings.Join(normalized, "\n\n"))
		question.Answer = answer
		index := fmt.Sprintf("%02d", question.Index)
		response[index] = question.Answer
	}

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(response, "arxiv")
}

func answerQuestion(llm *openai.OpenAI, question string, context string) string {
	messages := []openai.Message{
		{
			Role:    "system",
			Content: context,
		},
		{
			Role:    "system",
			Content: "I always answer with one sentence.",
		},
		{
			Role:    "user",
			Content: question,
		},
	}

	resp := llm.GetCompletionShort(messages, "gpt-4-turbo")
	if len(resp.Choices) == 0 {
		panic("no choices in response from LLM")
	}

	return resp.Choices[0].Message.Content
}

func fetchQuestions() []*Question {
	url := fmt.Sprintf("%s/data/%s/arxiv.txt", os.Getenv("CENTRALA_BASEURL"), os.Getenv("AI_DEVS_KEY"))

	response, err := http.Get(url)
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	questions := []*Question{}

	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		q := strings.Split(line, "=")
		if len(q) != 2 {
			continue
		}

		index, err := strconv.Atoi(q[0])
		if err != nil {
			panic(err)
		}

		question := Question{
			Index: index,
			Text:  q[1],
		}

		questions = append(questions, &question)
	}

	return questions
}

func normalizeData(llm *openai.OpenAI, cache *redis.Client, data ScrapeResults) []string {
	results := []string{}

	for _, section := range data.Sections {
		fragments := []string{}

		fragments = append(fragments, section.Title)

		for _, text := range section.Paragraphs {
			fragments = append(fragments, text)
		}

		for _, audio := range section.Audio {
			transcript := transcriptAudio(llm, cache, audio)
			fragments = append(fragments, transcript)
		}

		for _, image := range section.Images {
			description := describeImage(llm, cache, image)
			fragments = append(fragments, description)
		}

		results = append(results, strings.Join(fragments, "\n"))
	}

	return results
}

func describeImage(llm *openai.OpenAI, cache *redis.Client, image Image) string {
	ctx := context.Background()

	imageUrl := fmt.Sprintf("%s/dane/%s", os.Getenv("CENTRALA_BASEURL"), image.Url)

	var description string
	description, err := cache.Get(ctx, imageUrl).Result()
	if err != nil {
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
						Text: "I describe what's on the image. I include the city name of where the photo it was taken, if I can.",
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
		completions := llm.GetImageCompletionShort(messages, "gpt-4o")

		if len(completions.Choices) == 0 {
			panic("no completions returned by LLM")
		}

		description = completions.Choices[0].Message.Content

		cache.Set(ctx, imageUrl, description, time.Hour)
	}

	return description
}

func transcriptAudio(llm *openai.OpenAI, cache *redis.Client, url string) string {
	ctx := context.Background()

	audioUrl := fmt.Sprintf("%s/dane/%s", os.Getenv("CENTRALA_BASEURL"), url)

	var transcript string
	transcript, err := cache.Get(ctx, audioUrl).Result()
	if err != nil {
		audioResponse, err := http.Get(audioUrl)
		fileContents, err := io.ReadAll(audioResponse.Body)
		if err != nil {
			panic(err)
		}

		transcript = llm.GetTranscription(fileContents, "whisper-1", "mp3")

		cache.Set(ctx, audioUrl, transcript, time.Hour)
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

	var section *Section
	isSectionExists := false
	doc.Find("h1, h2, div, p, figure, a").Each(func(i int, s *goquery.Selection) {
		if s.Is("h2") {
			section = &Section{}
			results.Sections = append(results.Sections, section)

			section.Title = s.Text()
			isSectionExists = true
		}

		if !isSectionExists {
			return
		}

		if s.Is("p") {
			section.Paragraphs = append(section.Paragraphs, s.Text())
		}

		if s.Is("figure") {
			image := Image{}
			img := s.Find("img")
			if src, exists := img.Attr("src"); exists {
				image.Url = src
			}
			caption := s.Find("figcaption").Text()
			image.Caption = caption

			section.Images = append(section.Images, image)
		}

		if s.Is("a") {
			link, exists := s.Attr("href")
			if exists && strings.Contains(link, ".mp3") {
				section.Audio = append(section.Audio, link)
			}
		}
	})

	return results
}
