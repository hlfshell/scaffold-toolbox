package aws

import (
	"strings"
)

var allServices = []string{
	"s3",
	"dynamodb",
	"sqs",
	"sns",
	"lambda",
	"iam",
	"sts",
	"secretsmanager",
	"ssm",
	"cloudformation",
	"cloudwatch",
	"logs",
	"events",
	"scheduler",
	"pipes",
	"states",
	"kinesis",
	"firehose",
	"athena",
	"glue",
	"emr",
	"opensearch",
	"ec2",
	"imds",
	"ecs",
	"eks",
	"batch",
	"autoscaling",
	"elbv2",
	"rds",
	"rds-data",
	"elasticache",
	"route53",
	"cloudfront",
	"cloudfront-keyvaluestore",
	"servicediscovery",
	"apigateway",
	"apigatewayv2",
	"appsync",
	"kms",
	"acm",
	"wafv2",
	"waf",
	"cognito-idp",
	"cognito-identity",
	"organizations",
	"account",
	"ses",
	"efs",
	"ecr",
	"transfer",
	"s3files",
	"appconfig",
	"codebuild",
	"resourcegroupstaggingapi",
	"backup",
}

/*
WithAll enables every MiniStack service known to this toolbox module.
*/
func WithAll() Option {
	return func(stack *Stack) {
		stack.addService(allServices...)
	}
}

/*
WithServices enables raw MiniStack service names. Prefer typed With<Service>
helpers when one exists; keep this for escape hatches and newly added
MiniStack services.
*/
func WithServices(services ...string) Option {
	return func(stack *Stack) {
		stack.addService(services...)
	}
}

/*
WithS3 enables S3 and optionally creates buckets after MiniStack starts.
*/
func WithS3(buckets ...string) Option {
	return func(stack *Stack) {
		stack.addService("s3")
		stack.buckets = append(stack.buckets, buckets...)
	}
}

/*
WithDynamoDB enables DynamoDB and optionally creates simple string-key
tables after MiniStack starts.
*/
func WithDynamoDB(tables ...DynamoDBTable) Option {
	return func(stack *Stack) {
		stack.addService("dynamodb")
		stack.tables = append(stack.tables, tables...)
	}
}

/*
WithSQS enables SQS and optionally creates queues after MiniStack starts.
*/
func WithSQS(queues ...string) Option {
	return func(stack *Stack) {
		stack.addService("sqs")
		stack.queues = append(stack.queues, queues...)
	}
}

/*
WithSNS enables SNS and optionally creates topics after MiniStack starts.
*/
func WithSNS(topics ...string) Option {
	return func(stack *Stack) {
		stack.addService("sns")
		stack.topics = append(stack.topics, topics...)
	}
}

/*
WithLambda enables Lambda.
*/
func WithLambda() Option { return enable("lambda") }

/*
WithIAM enables IAM.
*/
func WithIAM() Option { return enable("iam") }

/*
WithSTS enables STS.
*/
func WithSTS() Option { return enable("sts") }

/*
WithSecretsManager enables Secrets Manager and optionally creates string
secrets after MiniStack starts.
*/
func WithSecretsManager(secrets ...Secret) Option {
	return func(stack *Stack) {
		stack.addService("secretsmanager")
		stack.secrets = append(stack.secrets, secrets...)
	}
}

/*
WithSSM enables SSM and optionally creates parameters after MiniStack starts.
*/
func WithSSM(parameters ...Parameter) Option {
	return func(stack *Stack) {
		stack.addService("ssm")
		stack.params = append(stack.params, parameters...)
	}
}

/*
WithCloudFormation enables CloudFormation.
*/
func WithCloudFormation() Option { return enable("cloudformation") }

/*
WithCloudWatch enables CloudWatch metrics and alarms.
*/
func WithCloudWatch() Option { return enable("cloudwatch") }

/*
WithCloudWatchLogs enables CloudWatch Logs.
*/
func WithCloudWatchLogs() Option { return enable("logs") }

/*
WithEventBridge enables EventBridge and optionally creates event buses
after MiniStack starts.
*/
func WithEventBridge(buses ...EventBus) Option {
	return func(stack *Stack) {
		stack.addService("events")
		stack.buses = append(stack.buses, buses...)
	}
}

/*
WithEventBridgeScheduler enables EventBridge Scheduler.
*/
func WithEventBridgeScheduler() Option { return enable("scheduler") }

/*
WithPipes enables EventBridge Pipes.
*/
func WithPipes() Option { return enable("pipes") }

/*
WithStepFunctions enables Step Functions.
*/
func WithStepFunctions() Option { return enable("states") }

/*
WithKinesis enables Kinesis and optionally creates streams after MiniStack starts.
*/
func WithKinesis(streams ...KinesisStream) Option {
	return func(stack *Stack) {
		stack.addService("kinesis")
		stack.streams = append(stack.streams, streams...)
	}
}

/*
WithFirehose enables Kinesis Data Firehose.
*/
func WithFirehose() Option { return enable("firehose") }

/*
WithAthena enables Athena.
*/
func WithAthena() Option { return enable("athena") }

/*
WithGlue enables Glue.
*/
func WithGlue() Option { return enable("glue") }

/*
WithEMR enables EMR.
*/
func WithEMR() Option { return enable("emr") }

/*
WithOpenSearch enables OpenSearch.
*/
func WithOpenSearch() Option { return enable("opensearch") }

/*
WithEC2 enables EC2.
*/
func WithEC2() Option { return enable("ec2") }

/*
WithIMDS enables the EC2 instance metadata service.
*/
func WithIMDS() Option { return enable("imds") }

/*
WithECS enables ECS. Use WithDockerSocket or WithDockerNetwork when tasks
need MiniStack to create Docker containers.
*/
func WithECS() Option { return enable("ecs") }

/*
WithEKS enables EKS.
*/
func WithEKS() Option { return enable("eks") }

/*
WithBatch enables Batch.
*/
func WithBatch() Option { return enable("batch") }

/*
WithAutoScaling enables Auto Scaling.
*/
func WithAutoScaling() Option { return enable("autoscaling") }

/*
WithELBv2 enables ALB / ELBv2.
*/
func WithELBv2() Option { return enable("elbv2") }

/*
WithRDS enables RDS. Use WithDockerSocket or WithDockerNetwork when
database dataplanes should be backed by real containers.
*/
func WithRDS() Option { return enable("rds") }

/*
WithRDSData enables the RDS Data API.
*/
func WithRDSData() Option { return enable("rds-data") }

/*
WithElastiCache enables ElastiCache. Use WithDockerSocket or
WithDockerNetwork when cache dataplanes should be backed by real containers.
*/
func WithElastiCache() Option { return enable("elasticache") }

/*
WithRoute53 enables Route 53.
*/
func WithRoute53() Option { return enable("route53") }

/*
WithCloudFront enables CloudFront.
*/
func WithCloudFront() Option { return enable("cloudfront") }

/*
WithCloudFrontKeyValueStore enables the CloudFront KeyValueStore data plane.
*/
func WithCloudFrontKeyValueStore() Option { return enable("cloudfront-keyvaluestore") }

/*
WithServiceDiscovery enables Cloud Map service discovery.
*/
func WithServiceDiscovery() Option { return enable("servicediscovery") }

/*
WithAPIGateway enables API Gateway v1.
*/
func WithAPIGateway() Option { return enable("apigateway") }

/*
WithAPIGatewayV2 enables API Gateway v2.
*/
func WithAPIGatewayV2() Option { return enable("apigatewayv2") }

/*
WithAppSync enables AppSync.
*/
func WithAppSync() Option { return enable("appsync") }

/*
WithKMS enables KMS.
*/
func WithKMS() Option { return enable("kms") }

/*
WithACM enables ACM.
*/
func WithACM() Option { return enable("acm") }

/*
WithWAFv2 enables WAFv2.
*/
func WithWAFv2() Option { return enable("wafv2") }

/*
WithWAFClassic enables WAF Classic.
*/
func WithWAFClassic() Option { return enable("waf") }

/*
WithCognitoUserPools enables Cognito user pools.
*/
func WithCognitoUserPools() Option { return enable("cognito-idp") }

/*
WithCognitoIdentityPools enables Cognito identity pools.
*/
func WithCognitoIdentityPools() Option { return enable("cognito-identity") }

/*
WithOrganizations enables Organizations.
*/
func WithOrganizations() Option { return enable("organizations") }

/*
WithAccount enables Account.
*/
func WithAccount() Option { return enable("account") }

/*
WithSES enables SES.
*/
func WithSES() Option { return enable("ses") }

/*
WithEFS enables EFS.
*/
func WithEFS() Option { return enable("efs") }

/*
WithECR enables ECR.
*/
func WithECR() Option { return enable("ecr") }

/*
WithTransfer enables Transfer Family.
*/
func WithTransfer() Option { return enable("transfer") }

/*
WithS3Files enables the S3 Files compatibility service.
*/
func WithS3Files() Option { return enable("s3files") }

/*
WithAppConfig enables AppConfig.
*/
func WithAppConfig() Option { return enable("appconfig") }

/*
WithCodeBuild enables CodeBuild.
*/
func WithCodeBuild() Option { return enable("codebuild") }

/*
WithResourceGroupsTagging enables the Resource Groups Tagging API.
*/
func WithResourceGroupsTagging() Option { return enable("resourcegroupstaggingapi") }

/*
WithBackup enables Backup.
*/
func WithBackup() Option { return enable("backup") }

func enable(service string) Option {
	return func(stack *Stack) {
		stack.addService(service)
	}
}

func (s *Stack) addService(services ...string) {
	s.services = append(s.services, services...)
}

func joinServices(services []string) string {
	return strings.Join(uniqueServices(services), ",")
}

func uniqueServices(services []string) []string {
	seen := map[string]bool{}
	output := []string{}
	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" || seen[service] {
			continue
		}
		seen[service] = true
		output = append(output, service)
	}

	return output
}
