package s3

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
)

// Return an s3 session, given a config.S3Config
func S3Session(c config.S3Config) *s3.S3 {
	s3Config := &aws.Config{
		Endpoint:         aws.String(c.EndpointURL),
		Region:           aws.String(c.Region),
		S3ForcePathStyle: aws.Bool(true),
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, "")
	}
	sess := session.New(s3Config)
	return s3.New(sess)
}

// UploadFile uploads the file filename to the supplied bucket with the key filekey using the provided S3 session.
func UploadFile(session *s3.S3, bucket string, filename string, filekey string) (err error) {
	key := aws.String(filekey)

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Unable to open file %s for upload: %s", filename, err)
		return
	}

	_, err = session.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: aws.String(bucket),
		Key:    key,
	})
	if err != nil {
		fmt.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		return
	}

	return
}

func ListBucket(session *s3.S3, bucket string) (err error) {
	resp, err := session.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		log.Printf("Unable to list items in bucket %q, %v", bucket, err)
		return
	}

	for _, item := range resp.Contents {
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
	}

	return
}
