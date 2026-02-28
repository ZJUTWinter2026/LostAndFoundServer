package llm

import (
	"app/comm"
	"context"
	"sync"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

var (
	embedModel     embedding.Embedder
	embedModelOnce sync.Once
)

func GetEmbeddingModel() embedding.Embedder {
	embedModelOnce.Do(func() {
		cfg := comm.BizConf.Agent.Embedding
		em, err := openai.NewEmbedder(context.Background(), &openai.EmbeddingConfig{
			Model:      cfg.Model,
			Dimensions: &cfg.Dimension,
			APIKey:     cfg.APIKey,
			BaseURL:    cfg.BaseURL,
		})
		if err != nil {
			panic(err)
		}
		embedModel = em
	})
	return embedModel
}
