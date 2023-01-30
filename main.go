package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	s3Client, err := NewS3Client(context.Background(), "us-east-1")
	if err != nil {
		panic(err)
	}
	response, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		panic(err)
	}
	for _, bucket := range response.Buckets {
		fmt.Printf("bucket: %v\n", *bucket.Name)
	}

}
