package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/di"
	"woyteck.pl/ai_devs3/internal/openai"
)

type File struct {
	Name     string
	Contents string
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

	url := fmt.Sprintf("%s/dane/pliki_z_fabryki.zip", os.Getenv("CENTRALA_BASEURL"))
	reports, facts := fetchData(url)

	context := []string{}
	for _, fact := range facts {
		context = append(context, fact.Contents)
	}
	for _, report := range reports {
		context = append(context, report.Contents)
	}

	contextString := strings.Join(context, "\n")
	contextString = strings.ReplaceAll(contextString, "agorski", "agowski")

	answer := map[string]string{}
	for _, report := range reports {
		fmt.Println("REPORT:")
		fmt.Println(report)

		keywords := generateKeywords(llm, contextString, report.Contents)
		keywords = append(keywords, report.Name)

		answer[report.Name] = strings.Join(keywords, ", ")
		fmt.Println("KEYWORDS: " + strings.Join(keywords, ", "))
		time.Sleep(time.Second)
	}

	fmt.Println("")
	fmt.Println("")

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(answer, "dokumenty")
}

func generateKeywords(llm *openai.OpenAI, context string, report string) []string {
	systemMessage := `<instruction>
Dla raportu podanego przez użytkownika generuję listę słów kluczowych w formie mianownika (czyli np. "sportowiec", a nie "sportowcem", "sportowców" itp.).
Analizuję w tym celu treści raportu i faktów, łączę fakty i na podstawie wniosków generuję słowa kluczowe.
</instruction>

<rules>
Zwracam tylko te słowa kluczowe, nic więcej.
Każde słowo kluczowe w osobnej linii, bez myślników i numerów linii.
</rules>

<facts>
` + context + `
</facts>`

	messages := []openai.Message{
		{
			Role:    "system",
			Content: systemMessage,
		},
		{
			Role:    "user",
			Content: report,
		},
	}
	completions := llm.GetCompletionShort(messages, "gpt-4o")
	if len(completions.Choices) == 0 {
		panic("no completions returned by LLM")
	}

	results := []string{}
	for _, keyword := range strings.Split(completions.Choices[0].Message.Content, "\n") {
		word := strings.ToLower(strings.Trim(keyword, " "))
		if word != "" {
			results = append(results, word)
		}
	}

	return results
}

func fetchData(url string) ([]File, []File) {
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

	reports := []File{}
	facts := []File{}

	for _, f := range archive.File {
		if !strings.Contains(f.Name, ".txt") {
			continue
		}

		if !strings.Contains(f.Name, "report") && !strings.Contains(f.Name, "facts/") {
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

		if strings.Trim(string(fileContents), " \n") == "entry deleted" {
			continue
		}

		if strings.Contains(f.Name, "report") {
			reports = append(reports, File{Name: f.Name, Contents: string(fileContents)})
		}

		if strings.Contains(f.Name, "facts/") {
			facts = append(facts, File{Name: f.Name, Contents: string(fileContents)})
		}
	}

	return reports, facts
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
