package on_demand_dataset_metadata_file_handler

const (
	notificationEmailSubject = "On Demand Dataset: %s ready for REMS access request"
	notificationEmailBody    = `
The On Demand Dataset you requested: %[1]s is ready for REMS access request.

Go to https://%[2]s/apply-for?resource=%[1]s to create your access request application.

This is an automated email.
`
)

// TODO

//func SendEmailNotifications(ctx context.Context, datasetAccession string) error {
//
//
//}
//func SendUserEmailNotification(ctx context.Context, datasetAccession, recipient string) error {
//	if !emailRemsNotification {
//		return nil
//	}
//
//	ctx, span := observability.StartSpan(ctx, "sendEmailNotification", attribute.String("accession", datasetAccession))
//	defer span.End()
//
//	m := gomail.NewMsg()
//
//	if err := m.From(emailAddress); err != nil {
//		return err
//	}
//
//	// TODO get user email
//	if err := m.To(recipient); err != nil {
//		return err
//	}
//
//	m.Subject(fmt.Sprintf(notificationEmailSubject, datasetAccession))
//
//	m.SetBodyString("text/html", fmt.Sprintf(notificationEmailBody, datasetAccession, remsUrl))
//
//	client, err := gomail.NewClient(
//		emailSmtpHost,
//		gomail.WithPort(emailSmtpPort),
//		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
//		gomail.WithUsername(emailAddress),
//		gomail.WithPassword(emailPassword),
//	)
//	if err != nil {
//		return err
//	}
//	defer func() {
//		if err := client.Close(); err != nil {
//			span.Warn("failed to close stmp client", slog.Any("error", err))
//		}
//	}()
//
//	if err := client.DialAndSendWithContext(ctx, m); err != nil {
//		return fmt.Errorf("failed to send rems notification email: %w", err)
//	}
//
//	return nil
//}
