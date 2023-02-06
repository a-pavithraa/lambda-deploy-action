package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/urfave/cli/v2"
)

type LambdaDeployParams struct {
	FunctionName         string
	BucketName           string
	KeyName              string
	Region               string
	ZipFile              string
	RoleArn              string
	Memory               int
	Timeout              int
	EnvironmentVariables map[string]string
}

func Client(ctx context.Context, region string) (*lambda.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %s", err)
	}
	lambdaClient := lambda.NewFromConfig(cfg, func(o *lambda.Options) {
		o.Region = region
	})
	return lambdaClient, nil

}
func UpdateFunctionConfiguration(ctx context.Context, lambdaParams LambdaDeployParams, client lambda.Client) error {
	configInput := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaParams.FunctionName,
	}
	memory := int32(lambdaParams.Memory)
	timeout := int32(lambdaParams.Timeout)
	if memory > 0 {
		configInput.MemorySize = &memory
	}
	if timeout > 0 {
		configInput.Timeout = &timeout
	}
	if lambdaParams.EnvironmentVariables != nil {
		configInput.Environment = &types.Environment{
			Variables: lambdaParams.EnvironmentVariables,
		}
	}
	if !TrimAndCheckEmptyString(&lambdaParams.RoleArn) {
		configInput.Role = &lambdaParams.RoleArn
	}
	_, err := client.UpdateFunctionConfiguration(ctx, configInput)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func UpdateFunctionCode(ctx context.Context, lambdaParams LambdaDeployParams, client lambda.Client) error {
	log.Println("Updating Function Code----------")
	functionInput := &lambda.UpdateFunctionCodeInput{
		FunctionName: &lambdaParams.FunctionName,
	}

	if !TrimAndCheckEmptyString(&lambdaParams.BucketName) && TrimAndCheckEmptyString(&lambdaParams.KeyName) {
		functionInput.S3Bucket = &lambdaParams.BucketName
		functionInput.S3Key = &lambdaParams.KeyName

	}
	if !TrimAndCheckEmptyString(&lambdaParams.ZipFile) {

		contents, err := GetFunctionCodeFromZip(lambdaParams.ZipFile)
		if err != nil {
			log.Println(err)
			return err
		}
		functionInput.ZipFile = contents
	}
	client.UpdateFunctionCode(ctx, functionInput)

	return nil

}

func GetFunctionCodeFromZip(fileName string) ([]byte, error) {

	zipFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer zipFile.Close()
	zipFileInfo, _ := zipFile.Stat()
	fileSize := zipFileInfo.Size()
	fileBuffer := make([]byte, fileSize)
	zipFile.Read(fileBuffer)

	return fileBuffer, nil

}
func FunctionConfigUpdateWithRetry(ctx context.Context, lambdaParams LambdaDeployParams, client lambda.Client) error {
	// Not able to perform 2 updates in succession immediately . So retrying till it is successful
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	log.Println("Updating Function Configuration----")
	var err error
	defer cancel()
	for {
		err = UpdateFunctionConfiguration(ctx, lambdaParams, client)

		if err != nil {
			log.Println("Retrying after 2 seconds---")

			time.Sleep(2 * time.Second)
		} else {
			log.Println("Resource Updated successfully")
			break
		}
	}
	return err
}

func main() {
	fmt.Println(os.Args)

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
			&cli.StringFlag{
				Name:    "roleArn",
				Usage:   "Role to be used for Lambda",
				EnvVars: []string{"ROLE_ARN", "INPUT_ROLE_ARN"},
			},
			&cli.IntFlag{
				Name:    "memory",
				Usage:   "Memory limit",
				EnvVars: []string{"MEMORY", "INPUT_MEMORY"},
			},
			&cli.IntFlag{
				Name:    "timeout",
				Usage:   "Timeout for Lambda",
				EnvVars: []string{"TIMEOUT", "INPUT_TIMEOUT"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			lambdaParams := LambdaDeployParams{
				Region:       cCtx.String("region"),
				FunctionName: cCtx.String("functionName"),
				BucketName:   cCtx.String("s3Bucket"),
				KeyName:      cCtx.String("s3Key"),
				ZipFile:      cCtx.String("zipFile"),
				Memory:       cCtx.Int("memory"),
				Timeout:      cCtx.Int("timeout"),
				RoleArn:      cCtx.String("roleArn"),
			}
			lambdaClient, err := Client(context.Background(), lambdaParams.Region)
			environmentVariables := cCtx.String("environmentVariables")
			if TrimAndCheckEmptyString(&environmentVariables) {

				result := make(map[string]string)
				if err = json.Unmarshal([]byte(environmentVariables), &result); err != nil {
					fmt.Println(err)
					log.Fatal(err)
				}
				lambdaParams.EnvironmentVariables = result
			}
			if TrimAndCheckEmptyString(&lambdaParams.FunctionName) {
				return errors.New("Function Name cannot be empty")

			}

			if (TrimAndCheckEmptyString(&lambdaParams.BucketName) && TrimAndCheckEmptyString(&lambdaParams.KeyName)) || !TrimAndCheckEmptyString(&lambdaParams.ZipFile) {
				UpdateFunctionCode(context.Background(), lambdaParams, *lambdaClient)
			}

			FunctionConfigUpdateWithRetry(context.Background(), lambdaParams, *lambdaClient)

			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
func TrimAndCheckEmptyString(s *string) bool {
	*s = strings.TrimSpace(*s)
	return len(*s) == 0
}
