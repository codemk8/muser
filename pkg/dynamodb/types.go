package dynamo

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// User is the user schame in dynamoDB
type User struct {
	UserName string
	Salt     string
	Email    string
	Verified bool
	Created  int64
	Data     map[string]interface{}
}

type DynamoClient struct {
	table string
	svc   *dynamodb.DynamoDB
}
