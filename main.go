package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/urfave/cli/v2"
)

func NewS3Client(ctx context.Context, region string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %s", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = region

	})

	return s3Client, nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "region",
				Usage:   "AWS Region",
				EnvVars: []string{"REGION", "AWS_REGION"},
			},
			&cli.StringFlag{
				Name:    "s3-bucket",
				Usage:   "password for the mysql database",
				EnvVars: []string{"S3_BUCKET"},
			},
			&cli.StringFlag{
				Name:    "function-name",
				Usage:   "lambda function name",
				EnvVars: []string{"FUNCTION_NAME"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Println(cCtx.String("region"), " function_name=", cCtx.String("function-name"))
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	s3Client, err := NewS3Client(context.Background(), "us-east-1")
	if err != nil {
		log.Fatal(err)
	}
	response, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal(err)
	}
	for _, bucket := range response.Buckets {
		fmt.Printf("bucket: %v\n", *bucket.Name)
	}

}
