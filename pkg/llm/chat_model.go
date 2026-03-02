package llm

import (
	"app/comm"
	"context"
	"sync"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

var (
	chatModel     model.ToolCallingChatModel
	chatModelOnce sync.Once

	visionModel     model.ToolCallingChatModel
	visionModelOnce sync.Once
)

func GetChatModel() model.ToolCallingChatModel {
	chatModelOnce.Do(func() {
		cfg := comm.BizConf.Agent.LLM
		cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			Model:   cfg.Model,
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
		})
		if err != nil {
			panic(err)
		}
		chatModel = cm
	})
	return chatModel
}

func GetVisionModel() model.ToolCallingChatModel {
	visionModelOnce.Do(func() {
		cfg := comm.BizConf.Agent.VisionLLM
		cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			Model:   cfg.Model,
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
		})
		if err != nil {
			panic(err)
		}
		visionModel = cm
	})
	return visionModel
}
