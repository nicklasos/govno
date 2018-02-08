package main

import "fmt"

type AwsStorage struct {}

func (s *AwsStorage) Upload(bucket, file, dest string) error {
	fmt.Println("Upload", bucket, file, dest)

	return nil
}
