package broker

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

//go:generate mockgen -source=broker.go -destination=../mocks/message_broker_mock.go -package=mocks
type MessageBroker interface {
	PublishWithContext(ctx context.Context, exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error
}
