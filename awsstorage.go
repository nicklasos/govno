package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type AwsStorage struct{}

func (s *AwsStorage) Upload(db Database, filename, dest string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	conf := aws.Config{
		Region: aws.String(db.AwsRegion),
		Credentials: credentials.NewStaticCredentials(
			db.AwsId,
			db.AwsKey,
			"",
		),
	}
	sess := session.New(&conf)
	svc := s3manager.NewUploader(sess)

	result, err := svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(db.AwsBucket),
		Key:    aws.String(filepath.Base(filename)),
		Body:   file,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully uploaded %s to %s\n", filename, result.Location)

	return nil
}
