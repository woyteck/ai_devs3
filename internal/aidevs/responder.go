package aidevs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Responder struct {
	url string
	key string
}

func NewResponder(url string, key string) *Responder {
	return &Responder{
		url: url,
		key: key,
	}
}

type Answer struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer any    `json:"answer"`
}

func (r *Responder) SendAnswer(message any, taskName string) {
	req := Answer{
		Task:   taskName,
		ApiKey: r.key,
		Answer: message,
	}

	json, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(bytes.NewBuffer(json))
	response, err := http.Post(r.url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}
