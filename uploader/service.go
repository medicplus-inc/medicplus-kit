package uploader

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nfnt/resize"

	"github.com/google/go-cloud/blob"
	"github.com/google/go-cloud/blob/gcsblob"
	"github.com/google/go-cloud/blob/s3blob"
	"github.com/google/go-cloud/gcp"
)

// Service represents the uploader service
type Service struct {
	bucket     *blob.Bucket
	bucketName string
	url        string
}

// ConfigParams represents the configuration params for aws
type ConfigParams struct {
	Cloud     string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Token     string
	URL       string
}

// generateFilePath generate url for new uploaded file
func generateFilePath(fileName string) (string, error) {
	utcNow := time.Now().UTC()
	datetimeValue := utcNow.Format("2006_01_02__15_04_05 ")
	buff := make([]byte, 32)
	_, err := rand.Read(buff)
	if err != nil {
		return "", err
	}

	hexString := fmt.Sprintf("%x", buff)
	fileExtension := fileName[strings.LastIndex(fileName, ".")+1 : len(fileName)]
	return fmt.Sprintf("./file/image/%s__%d__%s.%s", datetimeValue, hexString, fileExtension), nil
}

// doDelete process to delete to bucket
func (s *Service) doDelete(ctx context.Context, url string) error {
	bucket := "l8ldiytwq83d8ckg"
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return err
	}
	svc := s3.New(sess)

	_, err = svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(url)})
	if err != nil {
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(url),
	})
	if err != nil {
		return err
	}

	return nil
}

// doUpload process to upload to bucket
func (s *Service) doUpload(ctx context.Context, fileBytes []byte, url string) error {
	before := func(asFunc func(interface{}) bool) error {
		req := &s3manager.UploadInput{}
		ok := asFunc(&req)
		if !ok {
			return errors.New("invalid s3 type")
		}
		req.ACL = aws.String("public-read")
		return nil
	}
	bw, err := s.bucket.NewWriter(ctx, url, &blob.WriterOptions{
		BeforeWrite: before,
	})
	if err != nil {
		return err
	}

	_, err = bw.Write(fileBytes)
	if err != nil {
		return err
	}

	if err = bw.Close(); err != nil {
		return err
	}

	return nil
}

func resizeImage(imgBytes []byte) ([]byte, error) {
	imageInfo, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	var imgConverted image.Image
	if imageInfo.Width < imageInfo.Height {
		imgConverted = resize.Resize(0, 1000, img, resize.Lanczos3)
	} else {
		if imageInfo.Width < 1000 {
			imgConverted = resize.Resize(uint((imageInfo.Width * 80 / 100)), 0, img, resize.Lanczos3)
		} else {
			imgConverted = resize.Resize(1000, 0, img, resize.Lanczos3)
		}
	}

	imgBuffer := new(bytes.Buffer)
	err = jpeg.Encode(imgBuffer, imgConverted, nil)
	if err != nil {
		return nil, err
	}

	if bytes.NewReader(imgBuffer.Bytes()).Size() > 1000000 { //1MB
		resizeImage(imgBuffer.Bytes())
	}

	return imgBuffer.Bytes(), nil
}

// Upload upload file
func (s *Service) Upload(ctx context.Context, fileBytes []byte, fileName string) (*File, error) {
	url, err := generateFilePath(fileName)
	if err != nil {
		return nil, err
	}

	if http.DetectContentType(fileBytes) != "image/jpeg" || http.DetectContentType(fileBytes) != "image/png" {
		if bytes.NewReader(fileBytes).Size() > 1000000 {
			fileBytes, err = resizeImage(fileBytes)
			if err != nil {
				return nil, err
			}
		}
	}

	err = s.doUpload(ctx, fileBytes, url)
	if err != nil {
		return nil, err
	}

	return &File{
		URL: fmt.Sprintf("%s/%s/%s", s.url, s.bucketName, url[2:len(url)]),
	}, nil
}

// NewService creates new uploader service
func NewService(bucket *blob.Bucket, bucketName string, url string) *Service {
	return &Service{
		bucket:     bucket,
		bucketName: bucketName,
		url:        url,
	}
}

// SetupBucket creates a connection to a particular cloud provider's blob storage.
func SetupBucket(ctx context.Context, config *ConfigParams) (*blob.Bucket, error) {
	switch config.Cloud {
	case "aws":
		return setupAWS(ctx, config)
	case "gcp":
		return setupGCP(ctx, config.Bucket)
	default:
		return nil, fmt.Errorf("invalid cloud provider: %s", config.Cloud)
	}
}

// setupGCP setupGCP return bucket
func setupGCP(ctx context.Context, bucket string) (*blob.Bucket, error) {
	// DefaultCredentials assumes a user has logged in with gcloud.
	// See here for more information:
	// https://cloud.google.com/docs/authentication/getting-started
	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}
	c, err := gcp.NewHTTPClient(gcp.DefaultTransport(), gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}
	// The bucket name must be globally unique.
	return gcsblob.OpenBucket(ctx, bucket, c, nil)
}

// setupAWS setupAWS return bucket
func setupAWS(ctx context.Context, config *ConfigParams) (*blob.Bucket, error) {
	c := &aws.Config{
		// Either hard-code the region or use AWS_REGION.
		Region: aws.String(config.Region),
		// credentials.NewEnvCredentials assumes two environment variables are
		// present:
		// 1. AWS_ACCESS_KEY_ID, and
		// 2. AWS_SECRET_ACCESS_KEY.
		// Credentials: credentials.NewEnvCredentials(),
		Credentials: credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			config.Token,
		),
	}
	s := session.Must(session.NewSession(c))
	return s3blob.OpenBucket(ctx, config.Bucket, s, nil)
}
