package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// SESSender delivers Messages via AWS SES.
type SESSender struct {
	client    *ses.Client
	fromEmail string
	fromName  string
}

// NewSESSender builds a Sender backed by AWS SES.
func NewSESSender(cfg config.AWSConfig) (*SESSender, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("email: load config: %w", err)
	}
	return &SESSender{
		client:    ses.NewFromConfig(awsCfg),
		fromEmail: cfg.SES.FromEmail,
		fromName:  cfg.SES.FromName,
	}, nil
}

// Send dispatches a Message via SES.
func (s *SESSender) Send(ctx context.Context, msg Message) error {
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	input := &ses.SendEmailInput{
		Source: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: msg.To,
		},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String(msg.Subject)},
			Body: &types.Body{
				Html: &types.Content{Data: aws.String(msg.HTML)},
				Text: &types.Content{Data: aws.String(msg.Text)},
			},
		},
	}
	_, err := s.client.SendEmail(ctx, input)
	return err
}
