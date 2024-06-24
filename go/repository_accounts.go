package main

import (
	"context"
	"fmt"
	"log"

	"github.com/plaid/plaid-go/plaid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccountsRepository interface {
	SaveFor(ctx context.Context, u *User, accounts []plaid.AccountBase) error
	ListAllForUser(ctx context.Context, u *User) ([]plaid.AccountBase, error)
}

type accRepo struct {
	mongoCli        *mongo.Client
	collNamePattern string
}

func NewAccountsRepository(c *mongo.Client) AccountsRepository {
	return &accRepo{c, "%s_accounts"}
}

func (a *accRepo) SaveFor(ctx context.Context, user *User, accounts []plaid.AccountBase) error {

	coll := a.mongoCli.Database(databaseName).Collection(fmt.Sprintf(a.collNamePattern, user.UID))

	var data []interface{}
	for _, a := range accounts {
		data = append(data, a)
	}

	_, err := coll.InsertMany(ctx, data)

	return err
}

func (a *accRepo) ListAllForUser(ctx context.Context, user *User) ([]plaid.AccountBase, error) {

	coll := a.mongoCli.Database(databaseName).Collection(fmt.Sprintf(a.collNamePattern, user.UID))

	crsr, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	defer crsr.Close(ctx)

	all := make([]plaid.AccountBase, 0)

	for crsr.Next(context.Background()) {
		var a plaid.AccountBase
		if err := crsr.Decode(&a); err != nil {
			log.Println(err)
		} else {
			all = append(all, a)
		}
	}

	return all, nil
}
