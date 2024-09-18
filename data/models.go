package data

import (
	"context"
	"errors"
	"log"
	"snippet-service/constants"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SnippetRepository struct {
	client *mongo.Client
}

func NewSnippetRepository(mongoClient *mongo.Client) *SnippetRepository {
	return &SnippetRepository{
		client: mongoClient,
	}
}

type Snippet struct {
	ID         string    `bson:"_id,omitempty" json:"id,omitempty"`
	UserId     string    `bson:"user_id,omitempty" json:"user_id,omitempty"`
	Title      string    `bson:"title" json:"title"`
	Snippet    string    `bson:"snippet" json:"snippet"`
	Type       string    `bson:"type" json:"type"`
	OutputType string    `bson:"output_type" json:"output_type"`
	Output     string    `bson:"output,omitempty" json:"output,omitempty"`
	ImagePath  string    `bson:"image,omitempty" json:"image,omitempty"`
	Source     string    `bson:"source,omitempty" json:"source,omitempty"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

func (repo *SnippetRepository) Insert(snippet Snippet) (string, error) {
	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)
	result, err := collection.InsertOne(context.TODO(), Snippet{
		UserId:     snippet.UserId,
		Title:      snippet.Title,
		Snippet:    snippet.Snippet,
		Type:       snippet.Type,
		Output:     snippet.Output,
		OutputType: snippet.OutputType,
		ImagePath:  snippet.ImagePath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		log.Println("Error inserting into snippets", err)
		return "", err
	}

	// Assert the InsertedID to be of type primitive.ObjectID
	var id string
	if objectID, ok := result.InsertedID.(primitive.ObjectID); ok {
		id = objectID.Hex() // Convert to string representation
	} else {
		log.Println("Error: InsertedID is not of type primitive.ObjectID")
		return "", errors.New("failed to retrieve snippet ID")
	}

	return id, nil
}

func (repo *SnippetRepository) All() ([]*Snippet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(context.TODO(), bson.D{}, opts)
	if err != nil {
		log.Println("finding all docs error:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var snippets []*Snippet

	for cursor.Next(ctx) {
		var item Snippet
		err := cursor.Decode(&item)
		if err != nil {
			log.Println("error decoding log into slice:", err)
		} else {
			snippets = append(snippets, &item)
		}
	}
	return snippets, nil
}

func (repo *SnippetRepository) GetOne(id string) (*Snippet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var Snippet Snippet
	singleResult := collection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&Snippet)
	if singleResult.Error() != "" {
		return nil, err
	}

	return &Snippet, nil
}

func (repo *SnippetRepository) DropCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	if err := collection.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func (repo *SnippetRepository) Update(snippet Snippet) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	objectId, err := primitive.ObjectIDFromHex(snippet.ID)
	if err != nil {
		return nil, err
	}

	updateResult, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectId},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "title", Value: snippet.Title},
				{Key: "snippet", Value: snippet.Snippet},
				{Key: "type", Value: snippet.Type},
				{Key: "output", Value: snippet.Output},
				{Key: "updated_at", Value: time.Now()},
			}},
		},
	)

	if err != nil {
		return nil, err
	}
	return updateResult, nil
}

func (repo *SnippetRepository) Aggregate(pipeline any) ([]*Snippet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	cursor, err := collection.Aggregate(
		ctx,
		pipeline,
	)
	if err != nil {
		log.Println("error executing aggregate pipline", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var snippets []*Snippet

	for cursor.Next(ctx) {
		var item Snippet

		err := cursor.Decode(&item)
		if err != nil {
			log.Println("error decoding item to snippet", err)
			return nil, err
		}
		snippets = append(snippets, &item)
	}

	return snippets, nil
}

func (repo *SnippetRepository) Exists(title string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	// Use a case-insensitive regular expression to match the title
	pattern := bson.M{"title": bson.M{"$regex": title, "$options": "i"}}

	// Find one document with the matching title
	result := collection.FindOne(ctx, pattern)

	// Check if a document was found
	if result.Err() != nil {
		if err := result.Err(); err != nil && err != mongo.ErrNoDocuments {
			return false, err
		}
	} else {
		return true, nil
	}

	return false, nil
}
