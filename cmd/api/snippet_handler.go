package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"snippet-service/data"
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
	log.Println("Starting CreateSnippet handler...")

	// Parse the multipart form data
	log.Println("Parsing multipart form...")
	err := r.ParseMultipartForm(15 << 20) // 15 MB max memory
	if err != nil {
		log.Println("Error parsing form data:", err)
		http.Error(w, "Unable to parse form data", http.StatusBadRequest)
		return
	}

	// Get text fields from the form
	title := r.FormValue("title")
	snippet := r.FormValue("snippet")
	snippetType := r.FormValue("type")
	outputType := r.FormValue("output_type")
	outputText := r.FormValue("output")
	log.Printf("Received form data - title: %s, snippet: %s, type: %s, output_type: %s", title, snippet, snippetType, outputType)

	var imagePath string = ""

	// Basic validation
	if title == "" || snippet == "" || snippetType == "" || outputType == "" {
		log.Println("Missing required fields")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Handle image upload if the outputType is "image"
	if outputType == "image" {
		log.Println("Handling image file upload...")
		file, header, err := r.FormFile("image")
		if err != nil {
			log.Println("Error reading image file:", err)
			http.Error(w, "Unable to read image file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Read file content into a buffer
		fileBuffer := bytes.NewBuffer(nil)
		if _, err = io.Copy(fileBuffer, file); err != nil {
			log.Println("Error reading image file into buffer:", err)
			http.Error(w, "Failed to read image file", http.StatusInternalServerError)
			return
		}

		// Initialize the S3 client
		log.Println("Initializing S3 client...")
		s3Client, err := app.initS3Client()
		if err != nil {
			log.Println("Error initializing S3 client:", err)
			http.Error(w, "Unable to initialize S3 client", http.StatusInternalServerError)
			return
		}

		// Define the S3 key (filename in the bucket)
		s3Key := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), filepath.Base(header.Filename))
		log.Printf("Generated S3 key: %s", s3Key)

		// Upload the file to S3
		log.Println("Uploading image to S3...")
		uploader := manager.NewUploader(s3Client)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String("snippet-outputs"),
			Key:         aws.String(s3Key),
			Body:        bytes.NewReader(fileBuffer.Bytes()),
			ContentType: aws.String(header.Header.Get("Content-Type")),
		})

		if err != nil {
			log.Println("Error uploading image to S3:", err)
			http.Error(w, "Unable to upload image to S3", http.StatusInternalServerError)
			return
		}

		// Build the S3 image URL
		imagePath = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", "snippet-outputs", s3Key)
		log.Printf("Image successfully uploaded. S3 URL: %s", imagePath)
	}

	// Prepare the snippet data
	log.Println("Preparing snippet data...")
	req := data.Snippet{
		Title:      title,
		Snippet:    snippet,
		Type:       snippetType,
		ImagePath:  imagePath,
		OutputType: outputType,
		Output:     outputText,
	}

	// Insert the snippet into the database
	log.Println("Inserting snippet into the database...")
	err = app.Models.Snippet.Insert(req)
	if err != nil {
		log.Println("Error inserting snippet:", err)
		app.errorJSON(w, err)
		return
	}
	log.Println("Snippet inserted successfully")

	// Send a success response
	log.Println("Sending success response...")
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Snippet saved successfully"
	app.writeJSON(w, http.StatusOK, payload)
	log.Println("CreateSnippet handler completed successfully.")
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

	log.Println("==Search Snippet Results==")
	log.Println(snippets)

	payload := jsonResponse{
		Error:   false,
		Message: "We've got the best snippets for you!",
		Data:    snippets,
	}
	app.writeJSON(w, http.StatusOK, payload)
}
