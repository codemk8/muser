package dynamo

import (
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/codemk8/muser/pkg/schema"
	"github.com/golang/glog"
)

type DynamoClient struct {
	table     string
	svc       *dynamodb.DynamoDB
	blacklist map[string]bool
}

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
	items := []schema.User{}

	// Unmarshal the Items field in the result value to the Item Go type.
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		glog.Warningf("failed to unmarshal Query result items: %v", err)
		return nil, err
	}
	// fmt.Printf("Query %d items in the table.\n", len(items))
	return &DynamoClient{table: table, svc: svc, blacklist: NewBlackListMap()}, nil
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

func (client DynamoClient) BadUserName(username string) bool {
	return client.blacklist[username]
}

// GetUser returns a user in the table, if the user does not exist,
// it does not return error, only the key is empty (UserName)
func (client DynamoClient) GetUser(user string) (*schema.User, error) {
	keyCond := expression.Key("user_name").Equal(expression.Value(user))
	proj := expression.NamesList(expression.Name("created"),
		expression.Name("profile"),
		expression.Name("secret"))
	var expr expression.Expression
	var err error
	expr, err = expression.NewBuilder().WithKeyCondition(keyCond).WithProjection(proj).Build()
	if err != nil {
		glog.Warningf("failed to create the expression, %v", err)
		return nil, err
	}
	input := dynamodb.QueryInput{
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(client.table),
		Limit:                     aws.Int64(1),
		ScanIndexForward:          aws.Bool(false), // by created order
	}

	result, err := client.svc.Query(&input)
	if err != nil {
		glog.Warningf("Error get item: %v\n", err)
		return nil, err
	}
	users := []schema.User{}
	if len(result.Items) == 0 {
		return nil, errors.New("user not found")
	}
	// items := Project{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		glog.Warningf("Failed to unmarshal Record, %v", err)
		return nil, err
	}
	return &users[0], nil
}

// convert user.Data to attributeValue for PutItem
func convertAttrib(user *schema.User) (map[string]*dynamodb.AttributeValue, error) {
	av, err := dynamodbattribute.Marshal(user.Profile)
	return map[string]*dynamodb.AttributeValue{"object": av}, err
}

func (client DynamoClient) AddNewUser(user *schema.User) error {
	profile, err := dynamodbattribute.MarshalMap(user.Profile)
	if err != nil {
		glog.Warningf("Error mashal profile %v", err)
		return err
	}
	secret, err := dynamodbattribute.MarshalMap(user.Secret)
	if err != nil {
		glog.Warningf("Error mashal secret %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"user_name": {
				S: aws.String(user.UserName),
			},
			"created": {
				N: aws.String(strconv.FormatInt(user.Created, 10)),
			},
			"profile": {
				M: profile,
			},
			"secret": {
				M: secret,
			},
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		TableName:              aws.String(client.table),
	}

	_, err = client.svc.PutItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			glog.Warningf("dynamodb put item error type %s: %v", aerr.Code(), aerr)
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
func (client DynamoClient) UpdateUserPass(user *schema.User) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":p": {
				S: aws.String(user.Secret.Salt),
			},
		},

		Key: map[string]*dynamodb.AttributeValue{
			"user_name": {
				S: aws.String(user.UserName),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("SET secret.salt = :p"),
		TableName:        aws.String(client.table),
	}

	_, err := client.svc.UpdateItem(input)
	if err != nil {
		glog.Warningf("Error updating item: %v.", err)
		return err
	}
	return nil
}

func (client DynamoClient) UpdateUserEmail(user *schema.User) error {
	/*
		input := &dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":p": {
					S: aws.String(user.Email),
				},
			},

			Key: map[string]*dynamodb.AttributeValue{
				"email": {
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
	*/
	return nil
}
