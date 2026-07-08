package aws

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

/*
DynamoDBTable describes a simple DynamoDB table to create. Keys are
string attributes; PartitionKey defaults to "id" when blank.
*/
type DynamoDBTable struct {
	Name         string
	PartitionKey string
	SortKey      string
}

/*
Secret describes a Secrets Manager string secret to create.
*/
type Secret struct {
	Name  string
	Value string
}

/*
Parameter describes an SSM parameter to put.
*/
type Parameter struct {
	Name  string
	Value string
	Type  string
}

/*
KinesisStream describes a Kinesis stream to create.
*/
type KinesisStream struct {
	Name       string
	ShardCount int32
}

/*
EventBus describes an EventBridge event bus to create.
*/
type EventBus struct {
	Name string
}

func (s *Stack) createDynamoDBTables(ctx context.Context) error {
	if len(s.tables) == 0 {
		return nil
	}

	client, err := s.DynamoDBClient(ctx)
	if err != nil {
		return err
	}

	for _, table := range s.tables {
		if table.Name == "" {
			return fmt.Errorf("dynamodb table name is required")
		}
		partitionKey := table.PartitionKey
		if partitionKey == "" {
			partitionKey = "id"
		}

		attributes := []dynamodbtypes.AttributeDefinition{{
			AttributeName: awssdk.String(partitionKey),
			AttributeType: dynamodbtypes.ScalarAttributeTypeS,
		}}
		keySchema := []dynamodbtypes.KeySchemaElement{{
			AttributeName: awssdk.String(partitionKey),
			KeyType:       dynamodbtypes.KeyTypeHash,
		}}
		if table.SortKey != "" {
			attributes = append(attributes, dynamodbtypes.AttributeDefinition{
				AttributeName: awssdk.String(table.SortKey),
				AttributeType: dynamodbtypes.ScalarAttributeTypeS,
			})
			keySchema = append(keySchema, dynamodbtypes.KeySchemaElement{
				AttributeName: awssdk.String(table.SortKey),
				KeyType:       dynamodbtypes.KeyTypeRange,
			})
		}

		_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName:            awssdk.String(table.Name),
			AttributeDefinitions: attributes,
			KeySchema:            keySchema,
			BillingMode:          dynamodbtypes.BillingModePayPerRequest,
		})
		if err != nil {
			return fmt.Errorf("failed to create dynamodb table %s: %w", table.Name, err)
		}
	}

	return nil
}

func (s *Stack) createSecrets(ctx context.Context) error {
	if len(s.secrets) == 0 {
		return nil
	}

	client, err := s.SecretsManagerClient(ctx)
	if err != nil {
		return err
	}

	for _, secret := range s.secrets {
		if secret.Name == "" {
			return fmt.Errorf("secret name is required")
		}
		_, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         awssdk.String(secret.Name),
			SecretString: awssdk.String(secret.Value),
		})
		if err != nil {
			return fmt.Errorf("failed to create secret %s: %w", secret.Name, err)
		}
	}

	return nil
}

func (s *Stack) createParameters(ctx context.Context) error {
	if len(s.params) == 0 {
		return nil
	}

	client, err := s.SSMClient(ctx)
	if err != nil {
		return err
	}

	for _, param := range s.params {
		if param.Name == "" {
			return fmt.Errorf("parameter name is required")
		}
		paramType := ssmtypes.ParameterTypeString
		if param.Type != "" {
			paramType = ssmtypes.ParameterType(param.Type)
		}
		_, err := client.PutParameter(ctx, &ssm.PutParameterInput{
			Name:      awssdk.String(param.Name),
			Value:     awssdk.String(param.Value),
			Type:      paramType,
			Overwrite: awssdk.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("failed to put parameter %s: %w", param.Name, err)
		}
	}

	return nil
}

func (s *Stack) createKinesisStreams(ctx context.Context) error {
	if len(s.streams) == 0 {
		return nil
	}

	client, err := s.KinesisClient(ctx)
	if err != nil {
		return err
	}

	for _, stream := range s.streams {
		if stream.Name == "" {
			return fmt.Errorf("kinesis stream name is required")
		}
		shards := stream.ShardCount
		if shards <= 0 {
			shards = 1
		}
		_, err := client.CreateStream(ctx, &kinesis.CreateStreamInput{
			StreamName: awssdk.String(stream.Name),
			ShardCount: awssdk.Int32(shards),
		})
		if err != nil {
			return fmt.Errorf("failed to create kinesis stream %s: %w", stream.Name, err)
		}
	}

	return nil
}

func (s *Stack) createEventBuses(ctx context.Context) error {
	if len(s.buses) == 0 {
		return nil
	}

	client, err := s.EventBridgeClient(ctx)
	if err != nil {
		return err
	}

	for _, bus := range s.buses {
		if bus.Name == "" {
			return fmt.Errorf("event bus name is required")
		}
		_, err := client.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
			Name: awssdk.String(bus.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to create event bus %s: %w", bus.Name, err)
		}
	}

	return nil
}
