package main

import (
	"context"
	"fmt"
	"log"

	"github.com/plaid/plaid-go/plaid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionsRepository interface {
	SaveFor(ctx context.Context, u *User, t []plaid.Transaction) error
	ListAllForUser(ctx context.Context, u *User) ([]plaid.Transaction, error)
}

type transRepo struct {
	mongoCli        *mongo.Client
	collNamePattern string
}

func NewTransactionsRepository(c *mongo.Client) TransactionsRepository {
	return &transRepo{c, "%s_transactions"}
}

func (t *transRepo) SaveFor(ctx context.Context, user *User, transactions []plaid.Transaction) error {

	coll := t.mongoCli.Database(databaseName).Collection(fmt.Sprintf(t.collNamePattern, user.UID))

	var data []interface{}
	for _, t := range transactions {
		data = append(data, t)
	}

	_, err := coll.InsertMany(ctx, data)

	return err
}

func (t *transRepo) ListAllForUser(ctx context.Context, user *User) ([]plaid.Transaction, error) {

	coll := t.mongoCli.Database(databaseName).Collection(fmt.Sprintf(t.collNamePattern, user.UID))

	crsr, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer crsr.Close(ctx)

	all := make([]plaid.Transaction, 0)

	for crsr.Next(context.Background()) {
		var t plaid.Transaction
		if err := crsr.Decode(&t); err != nil {
			log.Println(err)
		} else {
			all = append(all, t)
		}
	}

	return all, nil

}
