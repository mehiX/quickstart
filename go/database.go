package main

import (
	"context"
	"log"
	"time"

	"github.com/plaid/plaid-go/plaid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoUserCert = "X509-cert-1674131487416060503.pem"
	connStr       = "mongodb+srv://clusterdevopsexperts.lujoh.mongodb.net/myFirstDatabase?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=" + mongoUserCert
	databaseName  = "plaid-trans"
)

var mongoCli *mongo.Client

var users UserRepository
var accountsRepo AccountsRepository
var transactionsRepo TransactionsRepository

func init() {

	var err error

	mongoCli, err = mongo.NewClient(options.Client().ApplyURI(connStr))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Created MongoDB client")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = mongoCli.Connect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")
	//defer mongoCli.Disconnect(ctx)

	err = mongoCli.Ping(ctx, nil)
	if err != nil {
		log.Println("No connection to MongoDB")
		return
	}

	log.Println("Ping MongoDB successful")

	users = NewUserRepository(mongoCli)
	accountsRepo = NewAccountsRepository(mongoCli)
	transactionsRepo = NewTransactionsRepository(mongoCli)

}

func saveToDb(ctx context.Context, user *User, accounts []plaid.AccountBase, transactions []plaid.Transaction) error {

	log.Println("Saving response")

	if err := accountsRepo.SaveFor(ctx, user, accounts); err != nil {
		log.Println("Error saving accounts", err)
	} else {
		log.Println("Accounts inserted")
	}

	if err := transactionsRepo.SaveFor(ctx, user, transactions); err != nil {
		log.Println("Error saving transactions", err)
	} else {
		log.Println("Transactions saved")
	}

	return nil
}
