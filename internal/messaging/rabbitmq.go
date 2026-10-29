package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	return &RabbitMQ{
		Connection: conn,
		Channel:    channel,
	}, nil
}

func (r *RabbitMQ) Close() error {
	if r == nil {
		return nil
	}

	var closeErr error

	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			closeErr = err
		}
	}

	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	return closeErr
}

func (r *RabbitMQ) DeclareQueue(name string) (amqp.Queue, error) {
	return r.Channel.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
}

type CreateUserRequest struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type CreateUserResponse struct {
	UserID string `json:"user_id"`
	Error  string `json:"error,omitempty"`
}

func (r *RabbitMQ) CreateUser(
	ctx context.Context,
	request CreateUserRequest,
) (string, error) {
	channel, err := r.Connection.Channel()
	if err != nil {
		return "", fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	defer channel.Close()

	queue, err := channel.QueueDeclare(
		"users.create",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("declare users queue: %w", err)
	}

	replyQueue, err := channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("declare reply queue: %w", err)
	}

	messages, err := channel.Consume(
		replyQueue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("consume reply queue: %w", err)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal create user request: %w", err)
	}

	correlationID := fmt.Sprintf(
		"%s-%d",
		request.UserID,
		time.Now().UnixNano(),
	)

	err = channel.PublishWithContext(
		ctx,
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       replyQueue.Name,
			Body:          body,
			DeliveryMode:  amqp.Persistent,
		},
	)
	if err != nil {
		return "", fmt.Errorf("publish create user request: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for user creation response: %w", ctx.Err())

		case message, ok := <-messages:
			if !ok {
				return "", fmt.Errorf("RabbitMQ reply channel closed")
			}

			if message.CorrelationId != correlationID {
				continue
			}

			var response CreateUserResponse
			if err := json.Unmarshal(message.Body, &response); err != nil {
				return "", fmt.Errorf("decode create user response: %w", err)
			}

			if response.Error != "" {
				return "", fmt.Errorf("users service: %s", response.Error)
			}

			if response.UserID == "" {
				return "", fmt.Errorf("users service returned an empty user ID")
			}

			return response.UserID, nil
		}
	}
}
