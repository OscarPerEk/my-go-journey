package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"encoding/json"

	"github.com/dslipak/pdf"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v2"
)

const (
	apiEndpointChat   = "https://api.openai.com/v1/chat/completions"
	apiEndpointEmbedd = "https://api.openai.com/v1/embeddings"
	embeddingsPath    = "embeddings.json"
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
	File     string
	Created  time.Time
	RowStart int
	RowEnd   int
	Vector   []float64
	Content  string
}

type Embeddings struct {
	Created    time.Time
	Embeddings []Embedding
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
			"model": "text-embedding-ada-002",
			"input": message,
		}).
		Post(apiEndpointEmbedd)
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
			"max_tokens": 1000,
		}).
		Post(apiEndpointChat)
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
	jsonFile, _ := os.Open(embeddingsPath)
	defer jsonFile.Close()
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

func GetEmbeddingDistances(question string, embeddings []Embedding) []EmbeddingDistance {
	// Get embedding of question
	var embeddingResponse EmbeddingResponse = CallEmbedding(question)
	var questionEmbedding []float64 = embeddingResponse.Data[0].Embedding
	// Find the closest embedding to the question
	var distances []EmbeddingDistance
	for _, embedding := range embeddings {
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

func ReadPdf(path string) string {
	r, err := pdf.Open(path)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		log.Fatalf("Could not extract text from pdf: %v\n", err)
	}
	buf.ReadFrom(b)
	return buf.String()
}

func ReadText(path string) string {
	// Read text file
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		log.Fatalf("Failed to read file %v\n", err)
	}
	scanner := bufio.NewScanner(file)
	var content string
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}
	return content
}

func ReadFile(path string) string {
	ftype := strings.Split(path, ".")[1]
	if ftype == "pdf" {
		content := ReadPdf(path)
		return content
	} else if ftype == "txt" || ftype == "md" {
		return ReadText(path)
	}
	log.Fatal("File type not supported. Only pdf, txt and md are supported.")
	return ""
}

func ConvertFileToEmbeddings(path string) []Embedding {
	// Convert file to embeddings
	// Read file
	// Split file content into embeddings
	content := ReadFile(path)
	var embeddings []Embedding
	var rowStart int
	var rowEnd int
	var lines []string = strings.Split(content, "\n")
	for i := 0; i < len(lines); i += 200 {
		if i-50 >= 0 {
			rowStart = i - 50
		} else {
			rowStart = i
		}
		if i+250 <= len(lines) {
			rowEnd = i + 250
		} else {
			rowEnd = len(lines)
		}
		contentPart := strings.Join(lines[rowStart:rowEnd], "\n")
		var embeddingResponse EmbeddingResponse = CallEmbedding(contentPart)
		embeddings = append(
			embeddings,
			Embedding{
				File:     path,
				Created:  time.Now(),
				RowStart: rowStart,
				RowEnd:   rowEnd,
				Vector:   embeddingResponse.Data[0].Embedding,
				Content:  contentPart,
			},
		)
	}
	return embeddings
}

func SaveEmbeddings(embeddings []Embedding) {
	// Save embeddings to file
	var embeddingsToSave Embeddings = Embeddings{
		Created:    time.Now(),
		Embeddings: embeddings,
	}
	embeddingsJson, err := json.Marshal(embeddingsToSave)
	if err != nil {
		log.Fatalf("Failed to marshal embeddings to json %v\n", err)
	}
	err = os.WriteFile(embeddingsPath, embeddingsJson, 0644)
	if err != nil {
		log.Fatalf("Failed to write embeddings to file %v\n", err)
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
	load := false
	var embeddings []Embedding
	if load {
		embeddings = LoadEmbeddings().Embeddings
	} else {
		embeddings = ConvertFileToEmbeddings("driving_license.txt")
		SaveEmbeddings(embeddings)
	}
	embeddingDistances := GetEmbeddingDistances(question, embeddings)
	context := GetContext(embeddingDistances, 2)
	var response GptResponse = CallChatgptWithContext(question, context)
	WriteAnswerToFile(response)
}
