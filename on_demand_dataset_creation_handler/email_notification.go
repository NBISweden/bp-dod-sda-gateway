package on_demand_dataset_creation_handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	gomail "github.com/wneessen/go-mail"
	"go.opentelemetry.io/otel/attribute"
)

const (
	notificationEmailSubject = "On Demand Dataset: %s ready for REMS access request"
	notificationEmailBody    = `
The On Demand Dataset you requested: %[1]s is ready for REMS access request.

Go to https://%[2]s/apply-for?resource=%[1]s to create your access request application.

This is an automated email.
`
)

func sendEmailNotifications(ctx context.Context, datasetAccession string) error {
	if !emailRemsNotification {
		return nil
	}

	ctx, span := observability.StartSpan(ctx, "sendEmailNotifications", attribute.String("accession", datasetAccession))
	defer span.End()

	client, err := gomail.NewClient(
		emailSmtpHost,
		gomail.WithPort(emailSmtpPort),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(emailAddress),
		gomail.WithPassword(emailPassword),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			span.Warn("failed to close stmp client", slog.Any("error", err))
		}
	}()

	userEmails, err := database.ListUserEmailsByDatasetAccession(ctx, datasetAccession)
	if err != nil {
		return fmt.Errorf("failed to list users who requested dataset creation: %w", err)
	}

	// No erroring during loop, as we do not want failed delivery to stop attempt to deliver to other users
	for _, recipient := range userEmails {
		msg, err := buildEmail(datasetAccession, recipient)
		if err != nil {
			span.Warn("failed to build rems notification email", slog.String("recipient", recipient), slog.Any("error", err))
		}

		if err := client.DialAndSendWithContext(ctx, msg); err != nil {
			span.Warn("failed to send rems notification email", slog.String("recipient", recipient), slog.Any("error", err))
		}
	}

	return nil
}
func buildEmail(datasetAccession, recipient string) (*gomail.Msg, error) {
	m := gomail.NewMsg()

	if err := m.From(emailAddress); err != nil {
		return nil, err
	}

	if err := m.To(recipient); err != nil {
		return nil, err
	}

	m.Subject(fmt.Sprintf(notificationEmailSubject, datasetAccession))

	m.SetBodyString("text/html", fmt.Sprintf(notificationEmailBody, datasetAccession, remsUrl))

	return m, nil
}

func (dmfh *onDemandDatasatCreationHandler) SendUserEmailNotification(ctx context.Context, datasetAccession, recipient string) error {
	if !emailRemsNotification {
		return nil
	}

	ctx, span := observability.StartSpan(ctx, "SendUserEmailNotification",
		attribute.String("accession", datasetAccession),
		attribute.String("recipient", recipient),
	)
	defer span.End()

	client, err := gomail.NewClient(
		emailSmtpHost,
		gomail.WithPort(emailSmtpPort),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(emailAddress),
		gomail.WithPassword(emailPassword),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			span.Warn("failed to close stmp client", slog.Any("error", err))
		}
	}()

	msg, err := buildEmail(datasetAccession, recipient)
	if err != nil {
		return fmt.Errorf("failed to build rems notification email: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("failed to send rems notification email: %w", err)
	}

	return nil
}
