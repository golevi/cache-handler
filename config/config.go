package config

// Config options
type Config struct {
	Type string `json:"type,omitempty"`
	Host string `json:"host,omitempty"`

	Bypass []string `json:"bypass"`
	Expire int      `json:"expire"`
	Cookie []string `json:"cookie"`
}
