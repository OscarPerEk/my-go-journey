package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"encoding/json"

	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v2"
)

const (
	apiEndpoint = "https://api.openai.com/v1/chat/completions"
)

type Config struct {
	Key string `yaml:"openai_api_key"`
}

type OpenAIResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created string   `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func parse_json() OpenAIResponse {
	// Parse json from file
	jsonFile, _ := os.Open("response.json")
	byteContent, _ := io.ReadAll(jsonFile)
	var parsed_response OpenAIResponse
	json.Unmarshal(byteContent, &parsed_response)
	fmt.Printf("\nResponse parsed: \n%+v\n", parsed_response)
	return parsed_response
}

func main() {
	// Read api key from yaml
	yamlPath := filepath.Join(filepath.Dir("."), "config.yaml")
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read yaml %v\n", err)
	}
	var config Config
	err = yaml.Unmarshal(yamlContent, &config)
	if err != nil {
		log.Fatalf("Failed to unmarschal yaml %v\n", err)
	}

	// Call OpenAI API
	fmt.Println("Calling OpenAI API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(config.Key).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model":      "gpt-3.5-turbo",
			"messages":   []interface{}{map[string]interface{}{"role": "system", "content": "You are a helpful assistant."}},
			"max_tokens": 50,
		}).
		Post(apiEndpoint)
	if err != nil {
		log.Fatalf("Failed to send request %v\n", err)
	}
	fmt.Println(response.String())

	var parsed_response OpenAIResponse
	json.Unmarshal(response.Body(), &parsed_response)
	fmt.Printf("\nResponse parsed: \n%+v\n", parsed_response)
}
