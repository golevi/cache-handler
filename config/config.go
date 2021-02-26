package config

// Config options
type Config struct {
	Type string `json:"type,omitempty"`
	Host string `json:"host,omitempty"`

	Bypass []string `json:"bypass"`
	Method []string `json:"method"`
	Expire int      `json:"expire"`
	Cookie []string `json:"cookie"`
}
