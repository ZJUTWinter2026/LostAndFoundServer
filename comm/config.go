package comm

// BizConf 业务配置
var BizConf BizConfig

type BizConfig struct {
	Upload    UploadConfig    `json:"upload" yaml:"upload" mapstructure:"upload"`
	LLM       LLMConfig       `json:"llm" yaml:"llm" mapstructure:"llm"`
	Embedding EmbeddingConfig `json:"embedding" yaml:"embedding" mapstructure:"embedding"`
	Milvus    MilvusConfig    `json:"milvus" yaml:"milvus" mapstructure:"milvus"`
}

type UploadConfig struct {
	Dir       string `json:"dir" yaml:"dir" mapstructure:"dir"`                         // 上传目录
	BaseURL   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`          // 访问前缀
	MaxSizeMB int64  `json:"max_size_mb" yaml:"max_size_mb" mapstructure:"max_size_mb"` // 单文件最大 MB
}

type LLMConfig struct {
	BaseURL string `json:"base_url" yaml:"base_url" mapstructure:"base_url"` // API地址
	APIKey  string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`    // API密钥
	Model   string `json:"model" yaml:"model" mapstructure:"model"`          // 模型名称
}

type EmbeddingConfig struct {
	BaseURL  string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`   // API地址
	APIKey   string `json:"api_key" yaml:"api_key" mapstructure:"api_key"`      // API密钥
	Model    string `json:"model" yaml:"model" mapstructure:"model"`            // 模型名称
	Dimension int   `json:"dimension" yaml:"dimension" mapstructure:"dimension"` // 向量维度
}

type MilvusConfig struct {
	Address     string `json:"address" yaml:"address" mapstructure:"address"`           // Milvus地址
	Collection  string `json:"collection" yaml:"collection" mapstructure:"collection"`  // Collection名称
}
