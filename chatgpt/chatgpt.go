package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/dslipak/pdf"
	"github.com/go-resty/resty/v2"
)

const (
	apiEndpointChat   = "https://api.openai.com/v1/chat/completions"
	apiEndpointEmbedd = "https://api.openai.com/v1/embeddings"
	model             = "gpt-3.5-turbo"
)

func getEmbeddingsPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, "embeddings.json")
	return filePath
}

func getAnswerPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, "answer.md")
	return filePath
}

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
	// binaryPath, err := os.Executable()
	// if err != nil {
	// 	log.Fatalf("Failed to get binary path %v\n", err)
	// }
	// yamlPath := filepath.Join(filepath.Dir(binaryPath), "config.yaml")
	// yamlPath := filepath.Join(filepath.Dir("."), "config.yaml")
	// yamlContent, err := os.ReadFile(yamlPath)
	// if err != nil {
	// 	log.Fatalf("Failed to read yaml %v\n", err)
	// }
	// var config Config
	// err = yaml.Unmarshal(yamlContent, &config)
	// if err != nil {
	// 	log.Fatalf("Failed to unmarschal yaml %v\n", err)
	// }
	// return config.Key
	return ReadAPIKey()
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

	var parsedResponse EmbeddingResponse
	json.Unmarshal(response.Body(), &parsedResponse)
	return parsedResponse
}

func CallChatgpt(message string, system_content string) GptResponse {
	// Call OpenAI API
	fmt.Println("Calling ChatGpt API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(GetOpenAiKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": model,
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
	var parsed_response GptResponse
	json.Unmarshal(response.Body(), &parsed_response)
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
	jsonFile, _ := os.Open(getEmbeddingsPath())
	defer jsonFile.Close()
	byteContent, _ := io.ReadAll(jsonFile)
	var parsedResponse Embeddings
	json.Unmarshal(byteContent, &parsedResponse)
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
	if strings.Contains(path, "https:") {
		// Read file from url
		client := resty.New()
		response, err := client.R().Get(path)
		if err != nil {
			log.Fatalf("Failed to send request %v\n", err)
		}
		return response.String()
	}
	split := strings.Split(path, ".")
	if len(split) > 1 {
		ftype := split[len(split)-1]
		if ftype == "pdf" {
			content := ReadPdf(path)
			return content
		}
	}
	return ReadText(path)
}

func ConvertFileToEmbeddings(path string) ([]Embedding, error) {
	// Convert file to embeddings
	// Read file
	// Split file content into embeddings
	content := ReadFile(path)
	if strings.Contains(content, "#protected") {
		return nil, fmt.Errorf("File is protected")
	}
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
		if len(embeddingResponse.Data) > 0 {
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
	}
	return embeddings, nil
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
	err = os.WriteFile(getEmbeddingsPath(), embeddingsJson, 0644)
	if err != nil {
		log.Fatalf("Failed to write embeddings to file %v\n", err)
	}
}

func WriteAnswerToFile(response GptResponse, embedding Embedding) {
	// Write answer to file
	answer := "# Answer from " + model + "\n\n" + response.Choices[0].Message.Content +
		"\n\n# Matched Context: \nFile: " + embedding.File +
		"\nRow Start: " + fmt.Sprintf("%v", embedding.RowStart) +
		"\nRow End: " + fmt.Sprintf("%v", embedding.RowEnd) +
		"\n\n" + embedding.Content
	file, err := os.Create(getAnswerPath())
	if err != nil {
		log.Fatalf("Failed to create file %v\n", err)
	}
	defer file.Close()
	_, err = file.WriteString(answer)
	if err != nil {
		log.Fatalf("Failed to write to file %v\n", err)
	}
}

func EmbeddFile(path string, embeddingsChannel chan []Embedding) {
	fmt.Println("\nFound file: ", path)
	fmt.Println("\nCreating new embedding: ", path)
	newEmbeddings, err := ConvertFileToEmbeddings(path)
	// var err error = nil
	// var newEmbeddings []Embedding = []Embedding{}

	if err != nil {
		fmt.Printf("\nFailed to create embedding: %v\n: %v\n", path, err)
	} else {
		fmt.Println("\nSuccessfully created embedding")
		embeddingsChannel <- newEmbeddings
	}
}

func EmbeddFolder(path string, wg *sync.WaitGroup, embeddingsChannel chan []Embedding) {
	// Get information about the file or directory
	if strings.Contains(path, "https:") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EmbeddFile(path, embeddingsChannel)
		}()
		return
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error:", err)
	} else if fileInfo.Mode().IsRegular() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EmbeddFile(path, embeddingsChannel)
		}()
	} else if fileInfo.Mode().IsDir() {
		fmt.Println("Found folder")
		fmt.Println(path, "is a directory.")

		folder, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error loading folder: %v\n%v\n", path, err)
		}
		fileInfos, err := folder.Readdir(-1)
		folder.Close()
		if err != nil {
			fmt.Printf("Error reading folder: %v\n%v\n", path, err)
		}
		for _, fileInfo := range fileInfos {
			p := filepath.Join(path, fileInfo.Name())
			EmbeddFolder(p, wg, embeddingsChannel)
		}
	}
}

func StartEmbedding(path string) {
	var wg sync.WaitGroup
	var embeddingsChannel chan []Embedding = make(chan []Embedding)
	EmbeddFolder(
		path,
		&wg,
		embeddingsChannel,
	)
	var newEmbeddings []Embedding
	go func() {
		for r := range embeddingsChannel {
			newEmbeddings = append(newEmbeddings, r...)
		}
	}()
	wg.Wait()
	close(embeddingsChannel)
	// Get previous embeddings
	fmt.Println("\nLoading previous embeddings: ", getEmbeddingsPath())
	embeddings := LoadEmbeddings().Embeddings
	embeddings = append(embeddings, newEmbeddings...)
	fmt.Println("\nSaving embedddings")
	SaveEmbeddings(embeddings)
}

func StartChat(question string) {
	var embeddings []Embedding = LoadEmbeddings().Embeddings
	var embeddingDistances []EmbeddingDistance = GetEmbeddingDistances(question, embeddings)
	var context string = GetContext(embeddingDistances, 2)
	var response GptResponse = CallChatgptWithContext(question, context)
	fmt.Printf("Answer from %v \n\n%v", model, response.Choices[0].Message.Content)
	WriteAnswerToFile(response, embeddingDistances[0].Embedding)
}

func ReadAPIKey() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, ".api_key.txt")
	apiKeyBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read api key %v\n", err)
	}
	return string(apiKeyBytes)
}

func WriteAPIKey(apiKey string) {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, ".api_key.txt")
	err = os.WriteFile(filePath, []byte(apiKey), 0644)
	if err != nil {
		log.Fatalf("Failed to write api key %v\n", err)
	}
}

func main() {
	var embeddPath string
	var apiKey string
	flag.StringVar(&embeddPath, "embedd", "", "Embedd a file or folder")
	flag.StringVar(&apiKey, "key", "", "Add an api key to the system")
	flag.Parse()
	args := flag.Args()
	if embeddPath != "" {
		StartEmbedding(embeddPath)
	} else if apiKey != "" {
		WriteAPIKey(apiKey)
	} else {
		StartChat(args[0])
	}
}
