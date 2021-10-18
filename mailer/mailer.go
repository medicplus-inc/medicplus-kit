package mailer

//MailData container of mail data in order to send email
type MailData struct {
	From    string
	To      string
	Subject string
	Body    string
}

// Mailer handle sending email process
type Mailer interface {
	SendEmail(mailData MailData)
}
