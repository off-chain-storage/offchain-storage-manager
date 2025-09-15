package db

import "go.mongodb.org/mongo-driver/v2/mongo"

// IsDuplicateKey reports whether the error is a MongoDB duplicate key error.
func IsDuplicateKey(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
