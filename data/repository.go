package data

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	Insert(snippet Snippet) (string, error)
	All() ([]*Snippet, error)
	GetOne(id string) (*Snippet, error)
	Update(snippet Snippet) (*mongo.UpdateResult, error)
	Aggregate(pipeline any) ([]*Snippet, error)
	Exists(title string) (bool, error)
	DropCollection() error
}

type UserRepo interface {
	Insert(user User) (string, error)
	GetAll() ([]*User, error)
	GetByEmail(email string) (*User, error)
	GetOne(userEmail string) (*User, error)
	GetUserSnippets(userEmail string) ([]Snippet, error)
	Update(user User, updateMap bson.M) error
	Delete(id string) error
}
