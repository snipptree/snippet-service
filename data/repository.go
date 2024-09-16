package data

import "go.mongodb.org/mongo-driver/mongo"

type Repository interface {
	Insert(snippet Snippet) error
	All() ([]*Snippet, error)
	GetOne(id string) (*Snippet, error)
	Update(snippet Snippet) (*mongo.UpdateResult, error)
	Aggregate(pipeline any) ([]*Snippet, error)
	Exists(title string) (bool, error)
	DropCollection() error
}
