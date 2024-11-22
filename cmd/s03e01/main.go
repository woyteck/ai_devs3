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

	// fmt.Println("REPORTS:")
	// for _, report := range reports {
	// 	fmt.Println(report)
	// }

	// fmt.Println("")
	// fmt.Println("")

	// fmt.Println("FACTS:")
	// for _, fact := range facts {
	// 	fmt.Println(fact)
	// }

	context := []string{}
	for _, fact := range facts {
		context = append(context, fact.Contents)
	}

	answer := map[string]string{}
	for _, report := range reports {
		fmt.Println("GENERATING KEYWORDS FOR:")
		fmt.Println(report)

		keywords := generateKeywords(llm, strings.Join(context, "\n"), report.Contents)
		answer[report.Name] = strings.Join(keywords, ", ")
		fmt.Println("KEYWORDS:" + strings.Join(keywords, ", "))
		time.Sleep(time.Second)
	}

	responder, ok := container.Get("responder").(*aidevs.Responder)
	if !ok {
		panic("responder factory failed")
	}
	responder.SendAnswer(answer, "dokumenty")
}

func generateKeywords(llm *openai.OpenAI, context string, report string) []string {
	messages := []openai.Message{
		{
			Role:    "system",
			Content: context,
		},
		{
			Role: "system",
			Content: `Generuję listę słów kluczowych w formie mianownika (czyli np. "sportowiec", a nie "sportowcem", "sportowców" itp.).
Dodaję dodatkowe słowa kluczowe na podstawie swojej wiedzy.
Zwracam tylko te słowa kluczowe, nic więcej.
Każde słowo kluczowe w osobnej linii, bez myślników i numerów linii.`,
		},
		{
			Role:    "user",
			Content: report,
		},
	}
	completions := llm.GetCompletionShort(messages, "gpt-4o-mini")
	if len(completions.Choices) == 0 {
		panic("no completions returned by LLM")
	}

	results := []string{}
	for _, keyword := range strings.Split(completions.Choices[0].Message.Content, "\n") {
		results = append(results, strings.ToLower(strings.Trim(keyword, " ")))
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
