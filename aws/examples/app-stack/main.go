package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hlfshell/scaffold"
	"github.com/hlfshell/scaffold-toolbox/aws"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cloud, err := aws.NewStack("cloud", "latest",
		aws.WithS3("documents"),
		aws.WithSQS("jobs"),
		aws.WithECSCluster("app"),
		aws.WithECSService(aws.ECSService{
			Name:         "api",
			Cluster:      "app",
			Family:       "api",
			LaunchType:   aws.ECSLaunchTypeFargate,
			DesiredCount: 1,
			Containers: []aws.ECSContainer{{
				Name:       "api",
				Dockerfile: "./api.Dockerfile",
				Image:      "app/api:dev",
				Env: map[string]string{
					"QUEUE_NAME": "jobs",
				},
				Ports: []aws.ECSPort{{
					ContainerPort: 8080,
					HostPort:      18080,
				}},
			}},
		}),
		aws.WithECSRunTask(aws.ECSRunTask{
			Name:       "worker",
			Cluster:    "app",
			Family:     "worker",
			LaunchType: aws.ECSLaunchTypeFargate,
			Containers: []aws.ECSContainer{{
				Name:       "worker",
				Dockerfile: "./worker.Dockerfile",
				Image:      "app/worker:dev",
				Env: map[string]string{
					"QUEUE_NAME": "jobs",
				},
			}},
		}),
	)
	if err != nil {
		return err
	}

	stack := scaffold.NewStack("app", scaffold.WithServices(cloud))
	if err := stack.Create(ctx); err != nil {
		return err
	}
	defer stack.Cleanup(context.WithoutCancel(ctx))

	if err := cloud.UploadObjectString(ctx, "documents", "seed/readme.txt", "hello from scaffold"); err != nil {
		return err
	}

	queueURL, _ := cloud.QueueURL("jobs")
	fmt.Println("AWS endpoint:", cloud.HostEnv()["AWS_ENDPOINT_URL"])
	fmt.Println("API endpoint:", "http://127.0.0.1:18080")
	fmt.Println("Queue URL:", queueURL)

	return nil
}
