package dynamo

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// User is the user schame in dynamoDB
type User struct {
	UserName string
	Pass     string
	Data     map[string]interface{}
}

type DynamoClient struct {
	table string
	svc   *dynamodb.DynamoDB
}
