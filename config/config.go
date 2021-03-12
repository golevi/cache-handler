package config

import "regexp"

// Config options
type Config struct {
	Type   string `json:"type,omitempty"`
	Host   string `json:"host,omitempty"`
	Expire int    `json:"expire"`

	Bypass Bypass `json:"bypass"`

	CookieRegexp []*regexp.Regexp
}

// Bypass sets what should be bypassed by the cache.
type Bypass struct {
	Paths   []string `json:"paths"`
	Methods []string `json:"methods"`
	Cookies []string `json:"cookies"`
}
