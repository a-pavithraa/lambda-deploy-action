package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/urfave/cli/v2"
)

type LambdaDeployParams struct {
	FunctionName         string
	BucketName           string
	KeyName              string
	Region               string
	ZipFile              string
	EnvironmentVariables map[string]string
}

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

func uploadFileToS3(ctx context.Context, s3Client *s3.Client, lambdaParams LambdaDeployParams) error {
	zipFile, err := os.Open(lambdaParams.ZipFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipFileInfo, _ := zipFile.Stat()
	fileSize := zipFileInfo.Size()
	fileBuffer := make([]byte, fileSize)
	zipFile.Read(fileBuffer)
	s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &lambdaParams.BucketName,
		Key:    &lambdaParams.KeyName,
		Body:   bytes.NewReader(fileBuffer),
	})
	return nil
}

func main() {
	fmt.Println(os.Args)
	s3Client, err := NewS3Client(context.Background(), "us-east-1")
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "region",
				Usage:   "AWS Region",
				EnvVars: []string{"REGION", "AWS_REGION", "INPUT_AWS_REGION"},
			},
			&cli.StringFlag{
				Name:    "s3Bucket",
				Usage:   "S3 bucket name containing the binary",
				EnvVars: []string{"S3_BUCKET", "INPUT_S3_BUCKET"},
			},
			&cli.StringFlag{
				Name:    "s3Key",
				Usage:   "S3 Key",
				EnvVars: []string{"S3_KEY", "INPUT_S3_KEY"},
			},
			&cli.StringFlag{
				Name:    "functionName",
				Usage:   "lambda function name",
				EnvVars: []string{"FUNCTION_NAME", "INPUT_FUNCTION_NAME"},
			},
			&cli.StringFlag{
				Name:    "zipFile",
				Usage:   "Binary to be uploaded",
				EnvVars: []string{"ZIP_FILE", "INPUT_ZIP_FILE"},
			},
			&cli.StringFlag{
				Name:    "environmentVariables",
				Usage:   "Environment variables to be used for Lambda",
				EnvVars: []string{"ENVIRONMENT_VARIABLES", "INPUT_ENVIRONMENT_VARIABLES"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			lambdaParams := LambdaDeployParams{
				Region:       cCtx.String("region"),
				FunctionName: cCtx.String("functionName"),
				BucketName:   cCtx.String("s3Bucket"),
				KeyName:      cCtx.String("s3Key"),
				ZipFile:      cCtx.String("zipFile"),
			}
			str := cCtx.String("environmentVariables")
			result := make(map[string]string)
			if err = json.Unmarshal([]byte(str), &result); err != nil {
				fmt.Println(err)
				log.Fatal(err)
			}
			lambdaParams.EnvironmentVariables = result
			fmt.Println(lambdaParams)

			uploadFileToS3(context.Background(), s3Client, lambdaParams)
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

}
