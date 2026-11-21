package onebot11

// Config 声明实例连接与媒体交付配置。
type Config struct {
	Mode         string          `json:"mode,omitempty"`
	WSURL        string          `json:"ws_url,omitempty"`
	AccessToken  string          `json:"access_token,omitempty"`
	SelfID       string          `json:"self_id,omitempty"`
	MediaURLBase string          `json:"media_url_base,omitempty"`
	MediaDir     *MediaDirectory `json:"media_dir,omitempty"`
}

// MediaDirectory 映射 Adapter 与协议端可见的同一共享目录。
type MediaDirectory struct {
	Local  string `json:"local"`
	Remote string `json:"remote"` // 空值沿用 Local。
}
