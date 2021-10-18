package notif

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

// EmailMessage email message object
type EmailMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
	ReplyTo string `json:"replyTo"`
}

// EmailNotifier email notifier using payfazz messenger as library
type EmailNotifier struct {
	sesSession *ses.SES
}

// Notify Notify function are not implemented on email
func (en *EmailNotifier) Notify(message string) error {
	return errors.New("Notify function are not implemented on email")
}

// Send sending email
func (en *EmailNotifier) Send(ctx context.Context, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var message EmailMessage
	json.Unmarshal(bytes, &message)

	sesEmailInput := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{aws.String(message.To)},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Data: aws.String(message.Message)},
			},
			Subject: &ses.Content{
				Data: aws.String(message.Subject),
			},
		},
		Source: aws.String(message.From),
		ReplyToAddresses: []*string{
			aws.String(message.From),
		},
	}

	_, err = en.sesSession.SendEmail(sesEmailInput)
	if err != nil {
		return err
	}

	return nil
}

// NewEmailNotifier create new email notifier
func NewEmailNotifier(
	awsRegion string,
	awsAccessKeyID string,
	awsSecretAccessKey string,
	awsToken string,
) *EmailNotifier {
	awsSession := session.New(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, awsToken),
	})

	sesSession := ses.New(awsSession)

	return &EmailNotifier{
		sesSession: sesSession,
	}
}
