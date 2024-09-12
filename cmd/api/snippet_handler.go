package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"snippet-service/data"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SearchSnippetsResult struct {
	Total    int             `json:"total"`
	Snippets []*data.Snippet `json:"snippets"`
}

// Initialize your S3 client (use this once in the config or startup)
func (app *Config) initS3Client() (*s3.Client, error) {
	staticCredentials := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("REDACTED_AWS_ACCESS_KEY_ID", "REDACTED_AWS_SECRET_ACCESS_KEY", ""))
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"), config.WithCredentialsProvider(staticCredentials))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %v", err)
	}
	return s3.NewFromConfig(cfg), nil
}

func (app *Config) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	log.Println("starting CreateSnippet handler...")

	// Parse the multipart form data
	log.Println("parsing multipart form...")
	err := r.ParseMultipartForm(15 << 20) // 15 MB max memory
	if err != nil {
		log.Println("error parsing form data:", err)
		app.errorJSON(w, errors.New("error parsing form data, memory limit exceeded"), http.StatusBadRequest)
		return
	}

	// Get text fields from the form
	title := r.FormValue("title")
	snippet := r.FormValue("snippet")
	snippetType := r.FormValue("type")
	outputType := r.FormValue("output_type")
	outputText := r.FormValue("output")
	log.Printf("received form data - title: %s, snippet: %s, type: %s, output_type: %s", title, snippet, snippetType, outputType)

	var imagePath string = ""

	// Basic validation
	if title == "" || snippet == "" || snippetType == "" || outputType == "" {
		log.Println("missing required fields")
		app.errorJSON(w, errors.New("Missing required fields"), http.StatusBadRequest)
		return
	}

	// Check for existing snippet title (case-insensitive)
	lowerCaseTitle := strings.ToLower(title)
	exists, err := app.Models.Snippet.Exists(lowerCaseTitle)
	if err != nil {
		log.Println("error checking for existing snippet:", err)
		app.errorJSON(w, err)
		return
	}

	if exists {
		log.Println("snippet with title already exists")
		app.errorJSON(w, errors.New("a similar snippet already exists in our system. please try changing your snippet or title"), http.StatusBadRequest)
		return
	}

	// Handle image upload if the outputType is "image"
	if outputType == "image" {
		log.Println("Handling image file upload...")
		file, header, err := r.FormFile("image")
		if err != nil {
			log.Println("error reading image file:", err)
			app.errorJSON(w, errors.New("unable to read image file"), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read file content into a buffer
		fileBuffer := bytes.NewBuffer(nil)
		if _, err = io.Copy(fileBuffer, file); err != nil {
			log.Println("error reading image file into buffer:", err)
			app.errorJSON(w, errors.New("error reading image file into buffer"), http.StatusBadRequest)
			return
		}

		// Initialize the S3 client
		log.Println("initializing S3 client...")
		s3Client, err := app.initS3Client()
		if err != nil {
			log.Println("error initializing S3 client:", err)
			app.errorJSON(w, errors.New("unable to initialize S3 client"), http.StatusInternalServerError)
			return
		}

		// Define the S3 key (filename in the bucket)
		s3Key := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), filepath.Base(header.Filename))
		log.Printf("generated S3 key: %s", s3Key)

		// Upload the file to S3
		log.Println("uploading image to S3...")
		uploader := manager.NewUploader(s3Client)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String("snippet-outputs"),
			Key:         aws.String(s3Key),
			Body:        bytes.NewReader(fileBuffer.Bytes()),
			ContentType: aws.String(header.Header.Get("Content-Type")),
		})

		if err != nil {
			log.Println("error uploading image to S3:", err)
			app.errorJSON(w, errors.New("unable to upload image to S3"), http.StatusInternalServerError)
			return
		}

		// Build the S3 image URL
		imagePath = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", "snippet-outputs", s3Key)
		log.Printf("image successfully uploaded. S3 URL: %s", imagePath)
	}

	// Prepare the snippet data
	log.Println("preparing snippet data...")
	req := data.Snippet{
		Title:      title,
		Snippet:    snippet,
		Type:       snippetType,
		ImagePath:  imagePath,
		OutputType: outputType,
		Output:     outputText,
	}

	// Insert the snippet into the database
	log.Println("inserting snippet into the database...")
	err = app.Models.Snippet.Insert(req)
	if err != nil {
		log.Println("error inserting snippet:", err)
		app.errorJSON(w, err)
		return
	}
	log.Println("snippet inserted successfully")

	// Send a success response
	log.Println("sending success response...")
	var payload jsonResponse
	payload.Error = false
	payload.Message = "snippet saved successfully"
	app.writeJSON(w, http.StatusOK, payload)
	log.Println("createSnippet handler completed successfully.")
}

func (app *Config) SearchSnippet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		app.errorJSON(w, errors.New("please enter a search query"))
		return
	}

	searchStage := bson.D{
		{Key: "$search", Value: bson.D{
			{Key: "index", Value: "snippet_title"}, //index name
			{Key: "text", Value: bson.D{
				{Key: "query", Value: query},  // Pass the search query variable
				{Key: "path", Value: "title"}, // The path to search (e.g., "title")
				{Key: "fuzzy", Value: bson.D{
					{Key: "maxEdits", Value: 2},       // Max number of edits (fuzziness)
					{Key: "prefixLength", Value: 0},   // Prefix length for the search
					{Key: "maxExpansions", Value: 50}, // Max expansions for fuzzy matching
				}},
			}},
		}},
	}

	// Create a pipeline with the search stage
	pipeline := mongo.Pipeline{searchStage}

	snippets, err := app.Models.Snippet.Aggregate(pipeline)
	if err != nil {
		log.Println("Error executing search:", err)
		app.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	if len(snippets) == 0 {
		generatedSnippet, err := app.fetchGeneratedSnippet(query)
		if err != nil {
			log.Println("error fetching from OpenAI:", err)
			app.errorJSON(w, err, http.StatusInternalServerError)
			return
		}

		err = app.Models.Snippet.Insert(*generatedSnippet)
		if err != nil {
			log.Println("error inserting generated snippet:", err)
			app.errorJSON(w, err, http.StatusInternalServerError)
			return
		}

		snippets = append(snippets, generatedSnippet)
	}

	log.Println("==Search Snippet Results==")
	log.Println(snippets)

	responseMessage := "We've got the best snippets for you!"

	if len(snippets) == 0 {
		responseMessage = "No snippets found for your search, Please comeback later!"
	}

	payload := jsonResponse{
		Error:   false,
		Message: responseMessage,
		Data:    snippets,
	}
	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) fetchGeneratedSnippet(query string) (*data.Snippet, error) {
	apiKey := "REDACTED_OPENAI_API_KEY"

	openAIPayload := map[string]interface{}{
		"model": "gpt-4", // use the correct OpenAI model name
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful assistant that generates code snippets.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Generate a code snippet for the following query: %s. Your response should have only code, no additional text.", query),
			},
		},
	}
	payloadBytes, err := json.Marshal(openAIPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}
	defer resp.Body.Close()

	var openAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	err = json.NewDecoder(resp.Body).Decode(&openAIResponse)
	if err != nil {
		return nil, err
	}

	if len(openAIResponse.Choices) > 0 {
		generatedCode := openAIResponse.Choices[0].Message.Content

		newSnippet := data.Snippet{
			Title:      query,
			Snippet:    generatedCode,
			Type:       "public",
			OutputType: "text",
			Output:     "",
			Source:     "AI",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		return &newSnippet, nil
	} else {
		return nil, errors.New("no snippet available from OpenAI")
	}
}
