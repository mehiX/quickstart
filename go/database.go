package main

import (
	"context"
	"log"
	"time"

	"github.com/plaid/plaid-go/plaid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoUserCert = "X509-cert-1674131487416060503.pem"
	connStr       = "mongodb+srv://clusterdevopsexperts.lujoh.mongodb.net/myFirstDatabase?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=" + mongoUserCert
)

var mongoCli *mongo.Client

func init() {

	var err error

	mongoCli, err = mongo.NewClient(options.Client().ApplyURI(connStr))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Created MongoDB client")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()

	err = mongoCli.Connect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")
	//defer mongoCli.Disconnect(ctx)

	err = mongoCli.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Ping MongoDB successful")

}

func saveToDb(ctx context.Context, d plaid.GetTransactionsResponse) error {

	log.Println("Saving response")

	res, err := saveAccounts(ctx, d.Accounts)
	if err != nil {
		log.Println("Error saving accounts", err)
	} else {
		log.Println("Accounts inserted: ", len(res.InsertedIDs))
	}

	res, err = saveTransactions(ctx, d.Transactions)
	if err != nil {
		log.Println("Error saving transactions", err)
	} else {
		log.Println("Transactions inserted: ", len(res.InsertedIDs))
	}

	return nil
}

func saveAccounts(ctx context.Context, accounts []plaid.Account) (*mongo.InsertManyResult, error) {
	accountsCollection := mongoCli.Database("plaid-trans").Collection("accounts")

	var data []interface{}
	for _, a := range accounts {
		data = append(data, a)
	}

	return accountsCollection.InsertMany(ctx, data)
}

func saveTransactions(ctx context.Context, transactions []plaid.Transaction) (*mongo.InsertManyResult, error) {
	transactionsCollection := mongoCli.Database("plaid-trans").Collection("transactions")

	var data []interface{}
	for _, t := range transactions {
		data = append(data, t)
	}

	return transactionsCollection.InsertMany(ctx, data)
}

func fetchAllTransactions(ctx context.Context) ([]plaid.Transaction, error) {
	tc := mongoCli.Database("plaid-trans").Collection("transactions")

	curr, err := tc.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer curr.Close(context.Background())

	all := make([]plaid.Transaction, 0)

	for curr.Next(context.Background()) {
		var t plaid.Transaction
		if err := curr.Decode(&t); err != nil {
			log.Println(err)
		} else {
			all = append(all, t)
		}
	}

	return all, nil

}

func fetchAllAccounts(ctx context.Context) ([]plaid.Account, error) {
	ac := mongoCli.Database("plaid-trans").Collection("accounts")

	curr, err := ac.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer curr.Close(context.Background())

	all := make([]plaid.Account, 0)

	for curr.Next(context.Background()) {
		var a plaid.Account
		if err := curr.Decode(&a); err != nil {
			log.Println(err)
		} else {
			all = append(all, a)
		}
	}

	return all, nil
}
