package comm

// BizConf 业务配置
var BizConf BizConfig

type BizConfig struct {
	Upload UploadConfig `json:"upload" yaml:"upload" mapstructure:"upload"`
}

type UploadConfig struct {
	Dir       string `json:"dir" yaml:"dir" mapstructure:"dir"`                         // 上传目录
	BaseURL   string `json:"base_url" yaml:"base_url" mapstructure:"base_url"`          // 访问前缀
	MaxSizeMB int64  `json:"max_size_mb" yaml:"max_size_mb" mapstructure:"max_size_mb"` // 单文件最大 MB
}
