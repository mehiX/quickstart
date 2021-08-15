package main

import (
	"context"
	"log"
	"time"

	"github.com/plaid/plaid-go/plaid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const connStr = "mongodb+srv://clusterdevopsexperts.lujoh.mongodb.net/myFirstDatabase?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=X509-cert-1674131487416060503.pem"

var mongoCli *mongo.Client

func init() {

	var err error

	mongoCli, err = mongo.NewClient(options.Client().ApplyURI(connStr))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Created MongoDB client")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err = mongoCli.Connect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")
	defer mongoCli.Disconnect(ctx)

	err = mongoCli.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Ping MongoDB successful")

}

func saveToDb(ctx context.Context, d plaid.GetTransactionsResponse) error {

	if err := mongoCli.Connect(ctx); err != nil {
		return err
	}
	defer mongoCli.Disconnect(ctx)

	accountsCollection := mongoCli.Database("plaid-trans").Collection("accounts")

	for _, a := range d.Accounts {
		if _, err := accountsCollection.InsertOne(ctx, a); err != nil {
			log.Println(err)
		}
	}

	return nil
}
