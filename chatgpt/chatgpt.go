package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

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

type GptResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created string   `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"` // list of answers/choices
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string         `json:"model"`
	Usage map[string]int `json:"usage"`
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"` // answer
}

type Embedding struct {
	Created  time.Time
	File     string
	RowStart int
	RowEnd   int
	Vector   []float64
	Content  string
}

type Embeddings struct {
	Created    time.Time
	Updated    time.Time
	Embeddings []Embedding
}

func BuildEmbedding(
	embeddingResponse EmbeddingResponse,
	content string,
	file string,
	rowStart int,
	rowEnd int,
) Embedding {
	return Embedding{
		Created:  time.Now(),
		File:     "embeddings.json",
		RowStart: rowStart,
		RowEnd:   rowEnd,
		Vector:   embeddingResponse.Data[0].Embedding,
		Content:  content,
	}
}

func GetOpenAiKey() string {
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
	return config.Key
}

func CallEmbedding(message string) EmbeddingResponse {
	// Call OpenAI API
	fmt.Println("Calling Embedding API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(GetOpenAiKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": "text-embedding-3-small",
			"input": message,
		}).
		Post(apiEndpoint)
	if err != nil {
		log.Fatalf("Failed to send request %v\n", err)
	}
	fmt.Println(response.String())

	var parsedResponse EmbeddingResponse
	json.Unmarshal(response.Body(), &parsedResponse)
	fmt.Printf("\nResponse parsed: \n%+v\n", parsedResponse)
	return parsedResponse
}

func CallChatgpt(message string, system_content string) GptResponse {
	// Call OpenAI API
	fmt.Println("Calling OpenAI API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(GetOpenAiKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{"role": "system", "content": system_content},
				map[string]interface{}{"role": "user", "content": message},
			},
			"max_tokens": 50,
		}).
		Post(apiEndpoint)
	if err != nil {
		log.Fatalf("Failed to send request %v\n", err)
	}
	fmt.Println(response.String())

	var parsed_response GptResponse
	json.Unmarshal(response.Body(), &parsed_response)
	fmt.Printf("\nResponse parsed: \n%+v\n", parsed_response)
	return parsed_response
}

func CallChatgptWithoutContext(message string) GptResponse {
	system_content := "You are an helpful assistant. Your job is to answer the user's questions."
	return CallChatgpt(message, system_content)
}

func CallChatgptWithContext(message string, context string) GptResponse {
	system_content := "Based on the context provided, your job is to first cite the relevant" +
		"answer found in context. Explicitly state in which file the answer is found. Then summarize" +
		"the answer in your own words. Formulate yourself using mark down sytaxt so that your answer can" +
		"be copy pasted to a md file. Your context is:" + context
	return CallChatgpt(message, system_content)
}

func LoadEmbeddings() Embeddings {
	// Parse json response from file
	jsonFile, _ := os.Open("embeddings.json")
	byteContent, _ := io.ReadAll(jsonFile)
	var parsedResponse Embeddings
	json.Unmarshal(byteContent, &parsedResponse)
	fmt.Printf("\nResponse parsed: \n%+v\n", parsedResponse)
	return parsedResponse
}

type EmbeddingDistance struct {
	Embedding Embedding
	Distance  float64
}

func GetVectorDistance(vector1 []float64, vector2 []float64) float64 {
	// Calculate the distance between two vectors
	var distance float64
	for i := 0; i < len(vector1); i++ {
		distance += (vector1[i] - vector2[i]) * (vector1[i] - vector2[i])
	}
	return distance
}

func GetEmbeddingDistances(question string) []EmbeddingDistance {
	// Get embedding of question
	var embeddingResponse EmbeddingResponse = CallEmbedding(question)
	var questionEmbedding []float64 = embeddingResponse.Data[0].Embedding
	// Find the closest embedding to the question
	var embeddings Embeddings = LoadEmbeddings()
	var distances []EmbeddingDistance
	for _, embedding := range embeddings.Embeddings {
		distances = append(distances, EmbeddingDistance{embedding, GetVectorDistance(embedding.Vector, questionEmbedding)})
	}
	// Sort distances
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})
	return distances
}

func GetContext(embeddingDistances []EmbeddingDistance, n int) string {
	// Get the context of the n closest embeddings
	var context string
	for i := 0; i < n; i++ {
		context += fmt.Sprintf(
			"File: %s\nContent from row %v to row %v:\n%v",
			embeddingDistances[i].Embedding.File,
			embeddingDistances[i].Embedding.RowStart,
			embeddingDistances[i].Embedding.RowEnd,
			embeddingDistances[i].Embedding.Content,
		)
	}
	return context
}

func ConvertFileToEmbeddings(path string) Embeddings {
	// Convert file to embeddings
	// Read file
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to read file %v\n", err)
	}
	scanner := bufio.NewScanner(file)
	// Split file content into embeddings
	var embeddings Embeddings
	embeddings.Created = time.Now()
	embeddings.Updated = time.Now()
	embeddings.Embeddings = []Embedding{}
	var rowStart int = 0
	var rowEnd int = 0
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
}

func WriteAnswerToFile(response GptResponse) {
	// Write answer to file
	answer := response.Choices[0].Message.Content
	fmt.Printf("\nAnswer: %s\n", answer)
	file, err := os.Create("answer.md")
	if err != nil {
		log.Fatalf("Failed to create file %v\n", err)
	}
	defer file.Close()
	_, err = file.WriteString(answer)
	if err != nil {
		log.Fatalf("Failed to write to file %v\n", err)
	}
}

func main() {
	question := "What is the speed limit in Germany?"
	embeddingDistances := GetEmbeddingDistances(question)
	context := GetContext(embeddingDistances, 2)
	var response GptResponse = CallChatgptWithContext(question, context)
	WriteAnswerToFile(response)
}
