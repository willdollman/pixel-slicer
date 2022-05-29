package s3

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

// S3Client contains configuration and a client for an S3-compatible storage service
type S3Client struct {
	S3     *s3.S3
	Config S3Config
}

// S3Config contains configuration for an S3-compatible storage service
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	EndpointURL     string `mapstructure:"Endpoint"`
	Region          string
	Bucket          string
	Enabled         bool
}

// NewSession returns a new S3 session, given an S3Config
func NewClient(conf S3Config) (client *S3Client) {
	s3Config := &aws.Config{
		Endpoint:         aws.String(conf.EndpointURL),
		Region:           aws.String(conf.Region),
		S3ForcePathStyle: aws.Bool(true),
	}
	if conf.AccessKeyID != "" && conf.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(conf.AccessKeyID, conf.SecretAccessKey, "")
	}

	sess := session.New(s3Config)

	return &S3Client{
		S3:     s3.New(sess),
		Config: conf,
	}
}

// UploadFile uploads the file filename to the supplied bucket with the key filekey using the provided S3 session.
func (s *S3Client) UploadFile(filename string, filekey string) error {
	key := aws.String(filekey)

	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "Unable to open file '%s' for upload", filename)
	}

	// Print error if file is larger than a reasonable size
	fi, err := f.Stat()
	if err != nil {
		return errors.Wrapf(err, "Unable to stat file %s", filename)
	}
	if fi.Size() > 50*1_000_000 {
		fmt.Printf("  Uploading large file: %d MB\n", fi.Size()/1_000_000)
	}

	_, err = s.S3.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: aws.String(s.Config.Bucket),
		Key:    key,
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to upload data to %s/%s", s.Config.Bucket, filename)
	}

	return nil
}

func (s *S3Client) ListBucket() (err error) {
	resp, err := s.S3.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(s.Config.Bucket)})
	if err != nil {
		log.Printf("Unable to list items in bucket %q, %v", s.Config.Bucket, err)
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
