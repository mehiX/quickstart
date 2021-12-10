package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	UID             string    `bson:"uid"`
	AccessToken     string    `bson:"accessToken"`
	ItemID          string    `bson:"itemId"`
	TokenReceivedAt time.Time `bson:"tokenReceivedAt"`
}

type UserRepository interface {
	Save(ctx context.Context, user User) error
	FindByUID(ctx context.Context, uid string) (*User, error)
}

type userRepo struct {
	coll *mongo.Collection
}

func (ur *userRepo) Save(ctx context.Context, user User) error {

	_, err := ur.coll.InsertOne(ctx, user)

	return err

}

func (ur *userRepo) FindByUID(ctx context.Context, uid string) (*User, error) {

	var user User
	err := ur.coll.FindOne(ctx, bson.M{"uid": uid}).Decode(&user)

	return &user, err
}

func NewUserRepository(mongoCli *mongo.Client) UserRepository {
	return &userRepo{
		mongoCli.Database("plaid-trans").Collection("users"),
	}
}
