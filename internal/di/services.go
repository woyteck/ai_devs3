package di

import (
	"os"

	"github.com/mendableai/firecrawl-go"
	"woyteck.pl/ai_devs3/internal/openai"
	"woyteck.pl/ai_devs3/internal/qdrant"
)

var Services = map[string]ServiceFactoryFn{
	"openai": func(c *Container) any {
		return openai.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
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
}
