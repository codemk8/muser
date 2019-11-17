package dynamo

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// User is the user schame in dynamoDB
type User struct {
	UserName string                 `json:"user_name,omitempty"`
	Salt     string                 `json:"salt,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Verified bool                   `json:"verified,omitempty"`
	Created  int64                  `json:"created,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

type DynamoClient struct {
	table string
	svc   *dynamodb.DynamoDB
}
