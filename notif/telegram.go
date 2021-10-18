package notif

import (
	"context"
	"errors"
	"fmt"

	"github.com/medicplus-inc/medicplus-kit/client"
)

// TelegramNotifier telegram notifier
type TelegramNotifier struct {
	telegramClient client.GenericHTTPClient
	channelID      string
	secretToken    string
}

// Notify send message to registered username
func (tn *TelegramNotifier) Notify(message string) error {
	ctx := context.Background()
	path := fmt.Sprintf(`bot%s/sendMessage?chat_id=%s&text=%s`, tn.secretToken, tn.channelID, message)
	errClient := tn.telegramClient.CallClient(&ctx, path, "POST", nil, nil, false)
	if errClient != nil && errClient.Error != nil {
		errString := fmt.Sprintf("Error on notify to Telegram: %v", errClient)
		return errors.New(errString)
	}

	return nil
}

// Send this method is not implemented yet
func (tn *TelegramNotifier) Send(ctx context.Context, data interface{}) error {
	return errors.New("send function are not implemented on Telegram")
}

// NewTelegramNotifier create new telegram notifier
func NewTelegramNotifier(
	telegramClient client.GenericHTTPClient,
	channelID string,
	secretToken string,
) *TelegramNotifier {
	return &TelegramNotifier{
		telegramClient: telegramClient,
		channelID:      channelID,
		secretToken:    secretToken,
	}
}
