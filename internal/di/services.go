package di

import (
	"fmt"
	"os"

	"github.com/mendableai/firecrawl-go"
	"github.com/redis/go-redis/v9"
	"woyteck.pl/ai_devs3/internal/aidevs"
	"woyteck.pl/ai_devs3/internal/llama"
	"woyteck.pl/ai_devs3/internal/openai"
	"woyteck.pl/ai_devs3/internal/qdrant"
)

var Services = map[string]ServiceFactoryFn{
	"responder": func(c *Container) any {
		url := fmt.Sprintf("%s/report", os.Getenv("CENTRALA_BASEURL"))
		return aidevs.NewResponder(url, os.Getenv("AI_DEVS_KEY"))
	},
	"openai": func(c *Container) any {
		return openai.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
	},
	"llama": func(c *Container) any {
		return llama.NewLlama(os.Getenv("LOCAL_LLAMA_URL"))
	},
	"qdrant": func(c *Container) any {
		return qdrant.NewClient(os.Getenv("QDRANT_HOST"))
	},
	"scraper": func(c *Container) any {
		fc, err := firecrawl.NewFirecrawlApp(os.Getenv("FIRECRAWL_API_KEY"), "https://api.firecrawl.dev")
		if err != nil {
			panic(err)
		}

		return fc
	},
	"redis": func(c *Container) any {
		return redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_HOST"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})
	},
}
