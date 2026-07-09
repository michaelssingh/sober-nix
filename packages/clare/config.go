package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Autoplay         bool   `json:"autoplay"`
	Autoskip         bool   `json:"autoskip"`
	SkipFillers      bool   `json:"skip_fillers"`
	AnilistToken     string `json:"anilist_token"`
	MalToken         string `json:"mal_token"`
	PreferredMode    string `json:"preferred_mode"`    // sub, dub, dual
	PreferredQuality string `json:"preferred_quality"` // best, 1080p, 720p, etc.
}

func getDefaultConfig() Config {
	return Config{
		Autoplay:         true,
		Autoskip:         true,
		SkipFillers:      false,
		PreferredMode:    "dual",
		PreferredQuality: "best",
	}
}

func getConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "clare", "config.json")
}

func loadConfig() Config {
	path := getConfigPath()
	f, err := os.Open(path)
	if err != nil {
		return getDefaultConfig()
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return getDefaultConfig()
	}
	// Fallback empty values to default config
	if cfg.PreferredMode == "" {
		cfg.PreferredMode = "dual"
	}
	if cfg.PreferredQuality == "" {
		cfg.PreferredQuality = "best"
	}
	return cfg
}

func saveConfig(cfg Config) error {
	path := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}
