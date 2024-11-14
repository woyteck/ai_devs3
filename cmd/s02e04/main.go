package main

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type Message struct {
	Description string `json:"description"`
}

type Note struct {
	FileName string `json:"fileName"`
	Contents string `json:"contents"`
}

type Results struct {
	People   []string `json:"people"`
	Hardware []string `json:"hardware"`
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

	notes := fetchNotes(llm, cache)

	peopleNotes := []Note{}
	hardwareNotes := []Note{}
	for _, note := range notes {
		category := categorizeNote(llm, note.Contents)
		if category == "LUDZIE" {
			peopleNotes = append(peopleNotes, note)
		}
		if category == "HARDWARE" {
			hardwareNotes = append(hardwareNotes, note)
		}
	}

	sort.Slice(peopleNotes, func(i, j int) bool {
		return peopleNotes[i].FileName < peopleNotes[j].FileName
	})
	sort.Slice(hardwareNotes, func(i, j int) bool {
		return hardwareNotes[i].FileName < hardwareNotes[j].FileName
	})

	results := Results{
		People:   []string{},
		Hardware: []string{},
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("LUDZIE:")
	for _, note := range peopleNotes {
		results.People = append(results.People, note.FileName)
		fmt.Println(note.FileName)
	}

	fmt.Println("HARDWARE:")
	for _, note := range hardwareNotes {
		results.Hardware = append(results.Hardware, note.FileName)
		fmt.Println(note.FileName)
	}

	fmt.Println("")
	fmt.Println("")

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(results, "kategorie")
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

func fetchNotes(llm *openai.OpenAI, cache *redis.Client) []Note {
	ctx := context.Background()

	var notes []Note
	cacheKey := "notes_json3"
	cachedNotes, err := cache.Get(ctx, cacheKey).Result()
	if err != nil {
		fmt.Println("cache miss")

		url := fmt.Sprintf("%s/dane/pliki_z_fabryki.zip", os.Getenv("CENTRALA_BASEURL"))
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

		notes = collectNotes(llm, archive.File)
		notesJson, err := json.Marshal(notes)
		if err != nil {
			panic(err)
		}
		cache.Set(ctx, cacheKey, notesJson, time.Hour)
	} else {
		fmt.Println("cache hit")
		err = json.Unmarshal([]byte(cachedNotes), &notes)
		if err != nil {
			panic(err)
		}
	}

	return notes
}

func collectNotes(llm *openai.OpenAI, files []*zip.File) []Note {
	notes := []Note{}

	for _, f := range files {
		if !strings.Contains(f.Name, ".txt") && !strings.Contains(f.Name, ".mp3") && !strings.Contains(f.Name, ".png") {
			continue
		}

		if strings.Contains(f.Name, "facts/") {
			continue
		}

		file, err := f.Open()
		if err != nil {
			panic(err)
		}

		fileContents, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}

		if strings.Contains(f.Name, ".txt") {
			notes = append(notes, Note{FileName: f.Name, Contents: string(fileContents)})
		}

		if strings.Contains(f.Name, ".mp3") {
			transcription := llm.GetTranscription(fileContents, "whisper-1", "mp3")
			notes = append(notes, Note{FileName: f.Name, Contents: transcription})
		}

		if strings.Contains(f.Name, ".png") {
			encodedImage := base64.StdEncoding.EncodeToString(fileContents)
			messages := []openai.ImageMessage{
				{
					Role: "system",
					Content: []openai.Content{
						{
							Type: "text",
							Text: "I return text from given images. Nothing else.",
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
					},
				},
			}
			completions := llm.GetImageCompletionShort(messages, "gpt-4o")
			if len(completions.Choices) > 0 {
				notes = append(notes, Note{FileName: f.Name, Contents: completions.Choices[0].Message.Content})
			}
		}
	}

	return notes
}

func categorizeNote(llm *openai.OpenAI, note string) string {
	context := `Jestem klasyfikatorem notatek
Zwracam w odpowiedzi konkretne słowo jeśli notatka zawiera informację o:
- schwytanych ludziach: LUDZIE
- śladach obecności ludzi: LUDZIE
- naprawionych usterkach hardwarowych: HARDWARE
W pozostałych przypadkach zwracam słowo NIEISTOTNE
`

	messages := []openai.Message{
		{
			Role:    "system",
			Content: context,
		},
		{
			Role:    "user",
			Content: note,
		},
	}
	resp := llm.GetCompletionShort(messages, "gpt-4o")
	if len(resp.Choices) == 0 {
		panic("no choices in response from LLM")
	}

	result := resp.Choices[0].Message.Content

	if result == "LUDZIE" || result == "HARDWARE" {
		return result
	}

	return "NIEISTOTNE"
}
