package data

import (
	"context"
	"log"
	"snippet-service/constants"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbTimeout = 5 * time.Second

type UserRepository struct {
	client *mongo.Client
}

func NewUserRepository(mongoClient *mongo.Client) *UserRepository {
	return &UserRepository{
		client: mongoClient,
	}
}

// User is the structure which holds one user document from the database.
type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email         string             `bson:"email" json:"email"`
	FirstName     string             `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName      string             `bson:"last_name,omitempty" json:"last_name,omitempty"`
	ProfilePicUrl string             `bson:"profile_pic_url,omitempty" json:"profile_pic_url,omitempty"`
	Snippets      []string           `bson:"snippets" json:"snippets"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// GetAll returns a slice of all users, sorted by last name.
func (repo *UserRepository) GetAll() ([]*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	opts := options.Find().SetSort(bson.D{{Key: "last_name", Value: 1}})
	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*User
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			log.Println("Error decoding user:", err)
			continue
		}
		users = append(users, &user)
	}

	return users, nil
}

// GetByEmail returns one user by email.
func (repo *UserRepository) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	var user User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetOne returns one user by ID.
func (repo *UserRepository) GetOne(userEmail string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)

	var user User
	err := collection.FindOne(ctx, bson.M{"email": userEmail}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (repo *UserRepository) GetUserSnippets(userEmail string) ([]Snippet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userCollection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	snippetCollection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.SNIPPETS_COLLECTION)

	var user User
	err := userCollection.FindOne(ctx, bson.M{"email": userEmail}).Decode(&user)
	if err != nil {
		return nil, err
	}

	var snippets []Snippet

	for _, snippetId := range user.Snippets {
		objectId, err := primitive.ObjectIDFromHex(snippetId)
		if err != nil {
			log.Printf("invalid ObjectID for snippetId %s: %v", snippetId, err)
			continue
		}

		var snippet Snippet
		err = snippetCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&snippet)
		if err != nil {
			log.Printf("error finding snippet details snippetId %s: %v", snippetId, err)
			continue
		}
		snippets = append(snippets, snippet)
	}

	return snippets, nil
}

// Update updates one user document.
func (repo *UserRepository) Update(user User, updateMap bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	filter := bson.M{"_id": user.ID}
	updateMap["updated_at"] = time.Now()
	update := bson.M{
		"$set": updateMap,
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes one user document by ID.
func (repo *UserRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	return nil
}

// Insert inserts a new user document into the database and returns the inserted ID.
func (repo *UserRepository) Insert(user User) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	collection := repo.client.Database(constants.DATABASE_NAME).Collection(constants.USERS_COLLECTION)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	res, err := collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	// Return the inserted ID as a string
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}
