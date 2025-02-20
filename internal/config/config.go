package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
)

type Config struct {
	DB_URL		string	`json:"db_url"`
	CurrentUser	string	`json:"current_user_name"`
}

const configName = ".gatorconfig.json"

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return path.Join(homeDir, configName)
}

func Read() (Config, error) {
	jsonFile, err := os.Open(getConfigPath())
	if err != nil {
		fmt.Println("Failed to load config file")
		return Config{}, err
	}

	defer jsonFile.Close()
	byteData, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Failed to read json file")
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(byteData, &config)
	if err != nil {
		fmt.Printf("Failed to unmarshal config. Error: %v\n", err)
		return Config{}, err
	}

	return config, nil
}

func SetUser(config Config) error {
	jsonData, err := json.Marshal(config)
	if err != nil {
		fmt.Printf("Failed to marshal the config. Error: %+v", err)
	}

	err = os.WriteFile(getConfigPath(), jsonData, 0666)
	if err != nil {
		fmt.Printf("Failed to write config. Error: %+v\n", err)
		return err
	}

	return nil
}
