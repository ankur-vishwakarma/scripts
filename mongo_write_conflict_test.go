package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestConflict(t *testing.T) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://127.0.0.1:27017"),
		options.Client().SetRetryWrites(false))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	collection := client.Database("enact").Collection("test-write-conflict")

	documentID := fmt.Sprintf("task%v", rand.Intn(1000))

	// Insert the initial document
	_, err = collection.InsertOne(context.TODO(), bson.M{"_id": documentID, "field": "initial value"})
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 1; i <= 2000; i++ { // till 1000 operations mongo queues the requests only after 1000 it starts throwing write conflict error
		wg.Add(1)

		i := i

		// Simulate concurrent write operations
		go func() {

			defer wg.Done()
			val := fmt.Sprintf("value:%v", i)
			err := updateDocumentWithTransaction(client, documentID, val)
			//fmt.Println("Updated document to:", val)
			handleWriteConflictError(err)
		}()

	}
	wg.Wait()
}

func updateDocumentWithTransaction(client *mongo.Client, documentID string, updatedValue string) error {
	session, err := client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.TODO())

	_, err = session.WithTransaction(context.TODO(), func(sessionContext mongo.SessionContext) (interface{}, error) {
		//time.Sleep(2000 * time.Millisecond)
		return nil, replaceDocument(sessionContext, client, documentID, updatedValue)
	})
	return err
}

func replaceDocument(ctx context.Context, client *mongo.Client, documentID string, updatedValue string) error {
	collection := client.Database("enact").Collection("test-write-conflict")

	filter := bson.M{"_id": documentID}
	replacement := bson.M{"field": updatedValue}

	_, err := collection.ReplaceOne(ctx, filter, replacement)
	return err
}

func handleWriteConflictError(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		if isWriteConflictError(err) {
			fmt.Println("Write conflict error:", err)
		} else {
			fmt.Println("Error:", err)
		}
	}
}

func isWriteConflictError(err error) bool {
	var mongoWriteException mongo.WriteException
	if errors.As(err, &mongoWriteException) {
		fmt.Println("this won't be printed:", err)
	}

	if mongoError, ok := err.(mongo.ServerError); ok {
		if mongoError.HasErrorCode(112) {
			fmt.Println("this works:", err)
		}
	}

	if writeException, ok := err.(mongo.WriteException); ok {
		fmt.Println("Error:", err)
		for _, writeError := range writeException.WriteErrors {
			if writeError.Code == 112 { // MongoDB error code for write conflict
				return true
			}
		}
	}
	return false
}
