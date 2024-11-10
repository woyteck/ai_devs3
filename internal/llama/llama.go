package llama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Llama struct {
	url string
}

type CompletionRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
	System string `json:"system,omitempty"`
}

type CompletionResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"createdAt"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason"`
	Context            []int     `json:"context"`
	TotalDuration      int       `json:"total_duration"`
	LoadDuration       int       `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int       `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int       `json:"eval_duration"`
}

func NewLlama(url string) *Llama {
	return &Llama{
		url: url,
	}
}

func (l *Llama) GetCompletion(request CompletionRequest) CompletionResponse {
	postBody, _ := json.Marshal(request)
	req, err := http.NewRequest("POST", l.url, bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("Error occured %v", err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error occured %v", err)
	}

	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal("Coult not read response")
		}
		fmt.Println(string(body))
	}

	var result CompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		log.Fatal("Can not unmarshall JSON")
	}

	return result
}

func (l *Llama) GetCompletionShort(prompt string, model string) CompletionResponse {
	request := CompletionRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	return l.GetCompletion(request)
}
