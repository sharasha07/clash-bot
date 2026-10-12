//go:build integration

package broker

import (
	"os"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestRabbitMQ(t *testing.T) {
	url, ok := os.LookupEnv("RABBITMQ_URL")
	require.True(t, ok)

	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = conn.Close()
		require.NoError(t, err)
	})

	ch, err := conn.Channel()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = ch.Close()
		require.NoError(t, err)
	})

	queue, err := ch.QueueDeclare(
		"test",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := ch.QueueDelete(queue.Name, false, false, false)
		require.NoError(t, err)
	})

	err = ch.PublishWithContext(t.Context(),
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("test_message"),
		},
	)
	require.NoError(t, err)
}
