package xfirebase_test

import (
	"context"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/raphoester/x/xconfig"
	"github.com/raphoester/x/xfirebase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Firebase xfirebase.Config `yaml:"firebase"`
}

func (c *Config) ResetToDefault() {
	c.Firebase.ResetToDefault()
}

func TestFirebaseCredentials(t *testing.T) {
	config := Config{}
	err := xconfig.NewFromDefaultFiles().ApplyConfig(&config)
	require.NoError(t, err)

	app, err := xfirebase.NewApp(context.Background(), config.Firebase)
	require.NoError(t, err)

	msgClient, err := app.Messaging(context.Background())
	require.NoError(t, err)

	_, err = msgClient.SendDryRun(context.Background(), &messaging.Message{
		Token:        "_invalid_token_",
		Notification: &messaging.Notification{Title: "title", Body: "body"},
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{Sound: "default"},
		},
		APNS: &messaging.APNSConfig{
			Payload:    &messaging.APNSPayload{Aps: &messaging.Aps{ThreadID: "tag"}},
			FCMOptions: &messaging.APNSFCMOptions{ImageURL: "https://image-url.com/image.jpg"},
		},
	})

	require.Error(t, err)

	// this assertion failing means that the credentials are invalid
	assert.Contains(t, err.Error(), "The registration token is not a valid FCM registration token")
}
