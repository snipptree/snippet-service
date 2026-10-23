package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"snippet-service/data"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongoURL is read from the MONGO_URL environment variable.
var mongoURL = os.Getenv("MONGO_URL")

const (
	webPort  = "80"
	rpcPort  = "5001"
	gRPCPort = "5001"
)

var client *mongo.Client

type Config struct {
	Repository data.Repository
	UserRepo   data.UserRepository
}

func main() {
	log.Printf("starting snippet service on PORT %s", webPort)
	if mongoURL == "" {
		log.Fatal("MONGO_URL must be set")
	}
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	app := Config{}
	app.setUpRepo(mongoClient)

	app.Serve()
}

func (app *Config) Serve() {
	srv := http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.Routes(),
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connectToMongo() (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(mongoURL)

	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("error connceting:", err)
		return nil, err
	}
	log.Println("connected to mongo db")
	return c, nil
}

func (app *Config) setUpRepo(client *mongo.Client) {
	repo := data.NewSnippetRepository(client)
	if repo == nil {
		log.Println("SnippetRepository is nil")
	}
	userRepo := data.NewUserRepository(client)
	app.Repository = repo
	app.UserRepo = *userRepo
}
