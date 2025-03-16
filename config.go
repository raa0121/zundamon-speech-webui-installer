package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type config struct {
	GitPath    string `toml:"GitPath"`
	PythonPath string `toml:"PythonPath"`
	IsCheck    bool   `toml:"IsCheck"`
}

func (cfg *config) load() error {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data")
	}
	configDir = filepath.Join(dir, "zundamon-speech-webui-installer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("cannot create directory: %v", err)
	}
	file := filepath.Join(configDir, "config.toml")

	_, err := os.Stat(file)
	if err == nil {
		// ファイルが存在している場合
		_, err := toml.DecodeFile(file, &cfg)
		if err != nil {
			return err
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func (cfg *config) save() error {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data")
	}
	configDir = filepath.Join(dir, "zundamon-speech-webui-installer")
	file := filepath.Join(configDir, "config.toml")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
