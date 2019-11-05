package dynamo

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/golang/glog"
)

// NewClient starts a new client
func NewClient(table string, region string) (*DynamoClient, error) {
	awscfg := &aws.Config{}
	awscfg.WithRegion(region)
	// Create the session that the DynamoDB service will use.
	sess := session.Must(session.NewSession(awscfg))

	// Create the DynamoDB service client to make the query request with.
	svc := dynamodb.New(sess)

	params := &dynamodb.ScanInput{
		TableName: aws.String(table),
		Limit:     aws.Int64(1), // limit for quick return
	}

	result, err := svc.Scan(params)
	if err != nil {
		glog.Warningf("Error db scanning: %v.", err)
		return nil, err
	}
	items := []User{}

	// Unmarshal the Items field in the result value to the Item Go type.
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		glog.Warningf("failed to unmarshal Query result items: %v", err)
		return nil, err
	}
	// fmt.Printf("Query %d items in the table.\n", len(items))
	return &DynamoClient{table: table, svc: svc}, nil
}

func (client DynamoClient) UserExist(user string) bool {
	item, err := client.GetUser(user)
	if err != nil {
		return false
	}
	if item.UserName == "" {
		return false
	}
	return true
}

// GetUser returns a user in the table, if the user does not exist,
// it does not return error, only the key is empty (UserName)
func (client DynamoClient) GetUser(user string) (*User, error) {
	result, err := client.svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(client.table),
		Key: map[string]*dynamodb.AttributeValue{
			"UserName": {
				S: aws.String(user),
			},
		},
	})
	if err != nil {
		glog.Warningf("Error get item: %v", err)
		return nil, err
	}
	item := User{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		glog.Warningf("Failed to unmarshal Record, %v", err)
		return nil, err
	}
	return &item, nil
}

// convert user.Data to attributeValue for PutItem
func convertAttrib(user *User) (map[string]*dynamodb.AttributeValue, error) {
	av, err := dynamodbattribute.Marshal(user.Data)
	return map[string]*dynamodb.AttributeValue{"object": av}, err
}

func (client DynamoClient) AddNewUser(user *User) error {
	attrib, err := convertAttrib(user)
	if err != nil {
		glog.Infof("Error converting attributes: %v.", err)
		return err
	}
	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"UserName": {
				S: aws.String(user.UserName),
			},
			"Email": {
				S: aws.String(user.Email),
			},
			"Salt": {
				S: aws.String(user.Salt),
			},
			"Created": {
				N: aws.String(strconv.FormatInt(user.Created, 10)),
			},
			"Data": {
				M: attrib,
			},
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		TableName:              aws.String(client.table),
	}

	_, err = client.svc.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeTransactionConflictException:
				fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			glog.Warningf("Error Put item in db: %v.", err)
		}
		return err
	}
	return nil
}

// UpdateUserPass updates the user password
func (client DynamoClient) UpdateUserPass(user *User) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":p": {
				S: aws.String(user.Salt),
			},
		},

		Key: map[string]*dynamodb.AttributeValue{
			"UserName": {
				S: aws.String(user.UserName),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET Salt = :p"),
		TableName:        aws.String(client.table),
	}

	_, err := client.svc.UpdateItem(input)
	if err != nil {
		glog.Warningf("Error updating item: %v.", err)
		return err
	}
	return nil
}

func (client DynamoClient) UpdateUserEmail(user *User) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":p": {
				S: aws.String(user.Email),
			},
		},

		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(user.Email),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET Email = :p"),
		TableName:        aws.String(client.table),
	}
	// TODO Verified flag needs to set to true
	_, err := client.svc.UpdateItem(input)
	if err != nil {
		glog.Warningf("Error updating item: %v", err)
		return err
	}
	return nil
}
