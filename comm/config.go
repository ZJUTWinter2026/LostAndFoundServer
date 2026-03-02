package comm

var BizConf BizConfig

type BizConfig struct {
	Upload    UploadConfig    `json:"upload" yaml:"upload" mapstructure:"upload"`
	Agent     AgentConfig     `json:"agent" yaml:"agent" mapstructure:"agent"`
}

type UploadConfig struct {
	Dir       string `json:"dir" yaml:"dir" mapstructure:"dir"`
	BaseURL   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	MaxSizeMB int64  `json:"max_size_mb" yaml:"max_size_mb" mapstructure:"max_size_mb"`
}

type AgentConfig struct {
	Enable    bool        `json:"enable" yaml:"enable" mapstructure:"enable"`
	LLM       LLMConfig   `json:"llm" yaml:"llm" mapstructure:"llm"`
	VisionLLM LLMConfig   `json:"vision_llm" yaml:"vision_llm" mapstructure:"vision_llm"`
	Embedding EmbedConfig `json:"embedding" yaml:"embedding" mapstructure:"embedding"`
	Milvus    MilvusConfig `json:"milvus" yaml:"milvus" mapstructure:"milvus"`
}

type LLMConfig struct {
	BaseURL string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	APIKey  string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	Model   string `json:"model" yaml:"model" mapstructure:"model"`
}

type EmbedConfig struct {
	BaseURL   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	APIKey    string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	Model     string `json:"model" yaml:"model" mapstructure:"model"`
	Dimension int    `json:"dimension" yaml:"dimension" mapstructure:"dimension"`
}

type MilvusConfig struct {
	Address    string `json:"address" yaml:"address" mapstructure:"address"`
	Collection string `json:"collection" yaml:"collection" mapstructure:"collection"`
}
